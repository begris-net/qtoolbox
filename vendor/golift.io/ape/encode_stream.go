package ape

import (
	"crypto/md5" //nolint:gosec // MD5 is the APE file checksum, not a security hash.
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"io"
	"math"
)

var errEncoderClosed = errors.New("ape: encoder closed")

// Encoder writes one APE file from PCM chunks.
// samples is the number of interleaved sample frames that will be written.
// Close patches the descriptor, header, and seek table, then writes the tag.
type Encoder struct {
	dst         io.WriteSeeker
	stream      Stream
	opt         *Options
	level       Compression
	align       int
	frameBlocks int
	samples     int
	frames      int
	frame       int
	accepted    int
	buf         []byte
	seek        []uint32
	carry       uint32
	carryLen    int
	sum         hash.Hash
	prefix      int
	closed      bool
}

// NewEncoder starts an APE file at the beginning of dst.
// samples must be the number of PCM frames Write will accept.
// A destination longer than the new file is not truncated.
func NewEncoder(dst io.WriteSeeker, stream Stream, samples int, opt *Options) (*Encoder, error) {
	level, frameBlocks, frames, err := encodePlan(stream, samples, opt)
	if err != nil {
		return nil, err
	}

	_, err = dst.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("ape: seeking start: %w", err)
	}

	prefix := descriptorSize + headerSize + frames*wordSize
	sum := md5.New() //nolint:gosec // MD5 is the APE file checksum, not a security hash.

	err = writeZeros(dst, prefix)
	if err != nil {
		return nil, err
	}

	err = writeExtra(dst, sum, opt.header())
	if err != nil {
		return nil, err
	}

	hold := min(samples, frameBlocks)

	return &Encoder{
		dst:         dst,
		stream:      stream,
		opt:         opt,
		level:       level,
		align:       stream.blockAlign(),
		frameBlocks: frameBlocks,
		samples:     samples,
		frames:      frames,
		buf:         make([]byte, 0, hold*stream.blockAlign()),
		seek:        make([]uint32, frames),
		sum:         sum,
		prefix:      prefix,
	}, nil
}

// Write accepts interleaved little-endian PCM.
// A partial trailing sample is buffered until the next Write or Close.
func (e *Encoder) Write(pcm []byte) error {
	if e.closed {
		return errEncoderClosed
	}

	if len(pcm) == 0 {
		return nil
	}

	if e.accepted+len(pcm) > e.samples*e.align {
		return errPCMLength
	}

	e.accepted += len(pcm)

	frameBytes := e.frameBlocks * e.align
	if len(e.buf) > 0 {
		need := min(frameBytes-len(e.buf), len(pcm))

		e.buf = append(e.buf, pcm[:need]...)
		pcm = pcm[need:]

		err := e.flushFull()
		if err != nil {
			return err
		}
	}

	for e.frame < e.frames-1 && len(pcm) >= frameBytes {
		err := e.emit(pcm[:frameBytes])
		if err != nil {
			return err
		}

		pcm = pcm[frameBytes:]
	}

	e.buf = append(e.buf, pcm...)

	return nil
}

// Close writes the last frame, the container header, and the APEv2 tag.
func (e *Encoder) Close() error {
	if e.closed {
		return errEncoderClosed
	}

	e.closed = true

	if e.accepted != e.samples*e.align || len(e.buf)%e.align != 0 {
		return errPCMLength
	}

	if len(e.buf) > 0 {
		err := e.emit(e.buf)
		if err != nil {
			return err
		}

		e.buf = nil
	}

	if e.frame != e.frames {
		return errPCMLength
	}

	var final [wordSize]byte

	// APE range decoders read one word past the last frame. A zero word is
	// still required when the payload already ends on a word boundary.
	if e.carryLen != 0 {
		binary.LittleEndian.PutUint32(final[:], e.carry)
	}

	_, err := e.dst.Write(final[:])
	if err != nil {
		return fmt.Errorf("ape: writing final word: %w", err)
	}

	e.sum.Write(final[:])

	return closeFile(e.dst, e.sum, e.seek, e.stream, e.level, e.opt, e.frameBlocks, e.samples, e.frames, e.prefix)
}

// flushFull encodes every full frame except the last. That frame stays
// buffered until Close, which is the only place that knows the write is finished.
func (e *Encoder) flushFull() error {
	frameBytes := e.frameBlocks * e.align

	for e.frame < e.frames-1 && len(e.buf) >= frameBytes {
		err := e.emit(e.buf[:frameBytes])
		if err != nil {
			return err
		}

		e.buf = e.buf[frameBytes:]
	}

	return nil
}

// emit compresses one frame and stitches the previous frame's leftover
// word onto it. Float samples are transformed on a copy. The unaligned
// tail is kept for the next frame; Close writes the terminating word.
func (e *Encoder) emit(pcm []byte) error {
	chunk := pcm
	if e.stream.Float {
		chunk = transformFloat(append([]byte(nil), pcm...))
	}

	payload := encodeFrame(chunk, e.stream.Channels, e.stream.Bits, e.level)

	pos, err := e.dst.Seek(0, io.SeekCurrent)
	if err != nil {
		return fmt.Errorf("ape: telling frame offset: %w", err)
	}

	e.seek[e.frame] = uint32(pos) + uint32(e.carryLen)

	body, word, ncarry := stitchFrame(payload, e.carry, e.carryLen)
	e.carry = word
	e.carryLen = ncarry
	aligned := len(body) / wordSize * wordSize

	_, err = e.dst.Write(body[:aligned])
	if err != nil {
		return fmt.Errorf("ape: writing frame: %w", err)
	}

	e.sum.Write(body[:aligned])
	e.frame++

	return nil
}

func encodePlan(stream Stream, samples int, opt *Options) (Compression, int, int, error) {
	if !stream.supported() {
		return 0, 0, 0, ErrUnsupported
	}

	if samples <= 0 {
		return 0, 0, 0, errPCMLength
	}

	if samples > math.MaxInt/stream.blockAlign() {
		return 0, 0, 0, ErrUnsupported
	}

	level := opt.compression()

	frameBlocks, err := opt.frameBlocks(level)
	if err != nil {
		return 0, 0, 0, err
	}

	frames, err := frameCount(samples, frameBlocks)
	if err != nil {
		return 0, 0, 0, err
	}

	finalBlocks := samples % frameBlocks
	if finalBlocks == 0 {
		finalBlocks = frameBlocks
	}

	err = checkFrames(level, frameBlocks, finalBlocks)
	if err != nil {
		return 0, 0, 0, err
	}

	return level, frameBlocks, frames, nil
}

// frameCount is the number of APE frames for samples, rounded up.
// The count has to fit the uint32 seek-table size in the descriptor.
func frameCount(samples, frameBlocks int) (int, error) {
	if frameBlocks <= 0 {
		return 0, ErrUnsupported
	}

	frames := samples / frameBlocks
	if samples%frameBlocks != 0 {
		frames++
	}

	prefix := descriptorSize + headerSize
	if frames <= 0 || frames > math.MaxUint32/wordSize || frames > (math.MaxInt-prefix)/wordSize {
		return 0, ErrUnsupported
	}

	return frames, nil
}

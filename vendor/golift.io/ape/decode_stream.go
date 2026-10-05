package ape

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// maxHeaderBytes is a ceiling for a descriptor or header.
// Real files use a 52-byte descriptor and a 24-byte header.
// A preserved legacy WAV header is skipped, not buffered, so it has no ceiling.
const maxHeaderBytes = 1 << 20

// Decoder yields interleaved PCM one frame at a time.
// The caller owns src and may close it after the decoder is finished.
type Decoder struct {
	src         io.ReadSeeker
	stream      Stream
	level       Compression
	version     int
	frames      int
	samples     int
	frameBlocks int
	finalBlocks int
	seek        []int64
	end         int64
	size        int64
	idx         int
}

// NewDecoder reads the APE header from src and prepares to decode frames.
// src is seeked to the start of the file. Versions before 3.93 are rejected.
func NewDecoder(src io.ReadSeeker) (*Decoder, error) {
	size, err := src.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, fmt.Errorf("ape: seeking end: %w", err)
	}

	_, err = src.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("ape: seeking start: %w", err)
	}

	if size < 8 {
		return nil, ErrUnsupported
	}

	var magic [8]byte

	_, err = io.ReadFull(src, magic[:])
	if err != nil {
		return nil, fmt.Errorf("ape: reading header: %w", err)
	}

	_, err = src.Seek(0, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("ape: seeking start: %w", err)
	}

	version := int(binary.LittleEndian.Uint16(magic[4:]))
	if string(magic[:4]) != magicMAC && string(magic[:4]) != magicMACF {
		return nil, ErrUnsupported
	}

	if version < version3980 {
		return parseOldDecoder(src, version, size)
	}

	return parseNewDecoder(src, size, magic[3] == 'F')
}

// Stream returns the PCM layout of the file.
func (d *Decoder) Stream() Stream { return d.stream }

// Samples returns the sample count stored in the header.
func (d *Decoder) Samples() int { return d.samples }

// Next returns the next frame of interleaved PCM.
// The final frame is shorter when the header says so.
// io.EOF is returned after the last frame.
func (d *Decoder) Next() ([]byte, error) {
	if d.idx >= d.frames {
		return nil, io.EOF
	}

	frame, skip, err := d.frameBytes(d.idx)
	if err != nil {
		return nil, err
	}

	blocks := d.frameBlocks
	if d.idx == d.frames-1 {
		blocks = d.finalBlocks
	}

	out, err := decodeFrame(frame, skip, blocks, d.stream, d.level, d.version, false)
	if err != nil {
		return nil, err
	}

	d.idx++

	if d.stream.Float {
		out = transformFloat(out)
	}

	return out, nil
}

func (d *Decoder) readAll() ([]byte, error) {
	size, err := pcmBytes(d.samples, d.stream.blockAlign())
	if err != nil {
		return nil, err
	}

	pcm := make([]byte, 0, size)

	for {
		chunk, err := d.Next()
		if errors.Is(err, io.EOF) {
			return pcm, nil
		}

		if err != nil {
			return nil, err
		}

		pcm = append(pcm, chunk...)
	}
}

func decodeSeeker(src io.ReadSeeker) ([]byte, Stream, error) {
	dec, err := NewDecoder(src)
	if err != nil {
		return nil, Stream{}, err
	}

	pcm, err := dec.readAll()
	if err != nil {
		return nil, Stream{}, err
	}

	return pcm, dec.stream, nil
}

func (d *Decoder) frameBytes(idx int) ([]byte, uint32, error) {
	start64 := d.seek[idx]
	if start64 < 0 || start64 > d.size {
		return nil, 0, errShortFrame
	}

	start := int(start64)
	remainder := int(start64-d.seek[0]) & (wordSize - 1)
	aligned := start - remainder

	stop64 := d.end
	if idx+1 < len(d.seek) {
		stop64 = d.seek[idx+1]
	}

	stop64 += wordSize
	if stop64 > d.size {
		stop64 = d.size
	}

	if aligned < 0 || int64(aligned) > stop64 {
		return nil, 0, errShortFrame
	}

	buf := make([]byte, int(stop64)-aligned)

	_, err := d.src.Seek(int64(aligned), io.SeekStart)
	if err != nil {
		return nil, 0, fmt.Errorf("ape: seeking frame: %w", err)
	}

	_, err = io.ReadFull(d.src, buf)
	if err != nil {
		return nil, 0, fmt.Errorf("ape: reading frame: %w", err)
	}

	return buf, uint32(remainder * 8), nil
}

//nolint:cyclop,funlen // the descriptor layout is one sequence of field checks
func parseNewDecoder(src io.ReadSeeker, size int64, floatMagic bool) (*Decoder, error) {
	lead, err := readN(src, 32)
	if err != nil {
		return nil, err
	}

	var (
		descBytes  = int(binary.LittleEndian.Uint32(lead[8:]))
		headBytes  = int(binary.LittleEndian.Uint32(lead[12:]))
		seekBytesN = int(binary.LittleEndian.Uint32(lead[16:]))
		headerData = int(binary.LittleEndian.Uint32(lead[20:]))
		frameBytes = int(binary.LittleEndian.Uint32(lead[24:])) +
			int(binary.LittleEndian.Uint32(lead[28:]))<<32
	)

	if descBytes < descriptorSize || descBytes > maxHeaderBytes ||
		headBytes < headerSize || headBytes > maxHeaderBytes {
		return nil, errShortDescriptor
	}

	if int64(descBytes)+int64(headBytes) > size {
		return nil, errShortDescriptor
	}

	rest, err := readN(src, descBytes+headBytes-len(lead))
	if err != nil {
		return nil, err
	}

	lead = append(lead, rest...)

	var (
		head        = lead[descBytes : descBytes+headBytes]
		level       = Compression(binary.LittleEndian.Uint16(head[0:]))
		flags       = binary.LittleEndian.Uint16(head[2:])
		frameBlocks = int(binary.LittleEndian.Uint32(head[4:]))
		finalBlocks = int(binary.LittleEndian.Uint32(head[8:]))
		frames      = int(binary.LittleEndian.Uint32(head[12:]))
		stream      = Stream{
			Bits:       int(binary.LittleEndian.Uint16(head[16:])),
			Channels:   int(binary.LittleEndian.Uint16(head[18:])),
			SampleRate: int(binary.LittleEndian.Uint32(head[20:])),
			Float:      floatMagic || flags&flagFloat != 0,
		}
	)

	err = checkFrames(level, frameBlocks, finalBlocks)
	if err != nil || !stream.supported() {
		return nil, ErrUnsupported
	}

	samples, err := sampleCount(frames, frameBlocks, finalBlocks)
	if err != nil {
		return nil, err
	}

	_, err = pcmBytes(samples, stream.blockAlign())
	if err != nil {
		return nil, err
	}

	if int64(frames)*wordSize > size || seekBytesN < frames*wordSize || int64(seekBytesN) > size {
		return nil, errEmptyAudio
	}

	seekRaw, err := readN(src, frames*wordSize)
	if err != nil {
		return nil, err
	}

	var (
		seek       = widenSeek(seekRaw, frames)
		audioStart = int64(descBytes + headBytes + seekBytesN)
		audioEnd   = min(audioStart+int64(headerData)+int64(frameBytes), size)
	)

	return &Decoder{
		src:         src,
		stream:      stream,
		level:       level,
		version:     int(binary.LittleEndian.Uint16(lead[4:])),
		frames:      frames,
		samples:     samples,
		frameBlocks: frameBlocks,
		finalBlocks: finalBlocks,
		seek:        seek,
		end:         audioEnd,
		size:        size,
	}, nil
}

//nolint:cyclop,funlen // the legacy header is one sequence of optional fields
func parseOldDecoder(src io.ReadSeeker, version int, size int64) (*Decoder, error) {
	if version < version3930 || size < oldHeaderSize {
		return nil, ErrUnsupported
	}

	raw, err := readN(src, oldHeaderSize)
	if err != nil {
		return nil, err
	}

	var (
		level        = Compression(binary.LittleEndian.Uint16(raw[6:]))
		flags        = binary.LittleEndian.Uint16(raw[8:])
		channels     = int(binary.LittleEndian.Uint16(raw[10:]))
		rate         = int(binary.LittleEndian.Uint32(raw[12:]))
		storedHeader = int(binary.LittleEndian.Uint32(raw[16:]))
		terminating  = int(binary.LittleEndian.Uint32(raw[20:]))
		frames       = int(binary.LittleEndian.Uint32(raw[24:]))
		finalBlocks  = int(binary.LittleEndian.Uint32(raw[28:]))
	)

	if frames <= 0 || int64(frames)*wordSize > size ||
		rate <= 0 || channels < 1 || channels > maxChannels {
		return nil, errEmptyAudio
	}

	if !level.known() || (version < version3950 && level == CompressionInsane) {
		return nil, ErrUnsupported
	}

	extra, seekCount, err := oldSeekCount(src, flags, frames, size)
	if err != nil {
		return nil, err
	}

	wav := 0
	if flags&flagCreateWAV == 0 {
		wav = storedHeader
	}

	if wav < 0 || int64(wav) > size {
		return nil, errShortDescriptor
	}

	seekAt := oldHeaderSize + extra + wav
	seekBytes := seekCount * wordSize

	if seekCount < frames || int64(seekAt)+int64(seekBytes) > size {
		return nil, errShortSeekTable
	}

	if wav > 0 {
		_, err = src.Seek(int64(wav), io.SeekCurrent)
		if err != nil {
			return nil, fmt.Errorf("ape: seeking wav header: %w", err)
		}
	}

	seekRaw, err := readN(src, frames*wordSize)
	if err != nil {
		return nil, err
	}

	frameBlocks := blocksFast
	if version >= version3950 {
		frameBlocks = blocksExtra
	}

	bits := 16

	switch {
	case flags&flag8Bit != 0:
		bits = 8
	case flags&flag24Bit != 0:
		bits = 24
	}

	err = checkFrames(level, frameBlocks, finalBlocks)
	if err != nil {
		return nil, err
	}

	stream := Stream{Bits: bits, Channels: channels, SampleRate: rate}

	samples, err := sampleCount(frames, frameBlocks, finalBlocks)
	if err != nil {
		return nil, err
	}

	_, err = pcmBytes(samples, stream.blockAlign())
	if err != nil {
		return nil, err
	}

	if !stream.supported() {
		return nil, ErrUnsupported
	}

	audioEnd := size - int64(terminating)
	if audioEnd < int64(seekAt) || audioEnd > size {
		audioEnd = size
	}

	return &Decoder{
		src:         src,
		stream:      stream,
		level:       level,
		version:     version,
		frames:      frames,
		samples:     samples,
		frameBlocks: frameBlocks,
		finalBlocks: finalBlocks,
		seek:        widenSeek(seekRaw, frames),
		end:         audioEnd,
		size:        size,
	}, nil
}

// oldSeekCount reads the optional peak level and seek-element count that sit
// immediately after the 32-byte legacy header. extra is how many bytes were read.
func oldSeekCount(src io.Reader, flags uint16, frames int, size int64) (int, int, error) {
	extra := 0
	if flags&flagHasPeak != 0 {
		extra += wordSize
	}

	seekCount := frames

	if flags&flagHasSeek == 0 {
		if extra == 0 {
			return 0, seekCount, nil
		}

		_, err := readN(src, extra)
		if err != nil {
			return 0, 0, err
		}

		return extra, seekCount, nil
	}

	at := oldHeaderSize + extra
	if int64(at+wordSize) > size {
		return 0, 0, errShortSeekCount
	}

	raw, err := readN(src, extra+wordSize)
	if err != nil {
		return 0, 0, err
	}

	seekCount = int(binary.LittleEndian.Uint32(raw[extra:]))
	extra += wordSize

	return extra, seekCount, nil
}

func readN(src io.Reader, n int) ([]byte, error) {
	if n < 0 {
		return nil, errShortDescriptor
	}

	buf := make([]byte, n)

	_, err := io.ReadFull(src, buf)
	if err != nil {
		return nil, fmt.Errorf("ape: reading header: %w", err)
	}

	return buf, nil
}

package ape

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Decode reads a version 3990 APE file, or a file from versions 3.93 through
// 3.99, and returns interleaved little-endian PCM.
// A seekable reader (io.ReadSeeker) positioned at the start is decoded a frame at a time.
func Decode(src io.Reader) ([]byte, Stream, error) {
	if rs, ok := src.(io.ReadSeeker); ok {
		pos, posErr := rs.Seek(0, io.SeekCurrent)
		if posErr == nil && pos == 0 {
			return decodeSeeker(rs)
		}
	}

	raw, err := io.ReadAll(src)
	if err != nil {
		return nil, Stream{}, fmt.Errorf("ape: reading file: %w", err)
	}

	file, err := parseFile(raw)
	if err != nil {
		return nil, Stream{}, err
	}

	size, err := pcmBytes(file.samples, file.stream.blockAlign())
	if err != nil {
		return nil, Stream{}, err
	}

	pcm := make([]byte, 0, size)

	for idx := range file.frames {
		frame, skip := file.frame(idx)

		chunk, err := decodeFrame(frame, skip, file.blocks(idx), file.stream, file.level, file.version, false)
		if err != nil {
			return nil, Stream{}, err
		}

		pcm = append(pcm, chunk...)
	}

	if file.stream.Float {
		pcm = transformFloat(pcm)
	}

	return pcm, file.stream, nil
}

type apeFile struct {
	stream  Stream
	level   Compression
	version int
	blocks  func(int) int
	frames  int
	samples int
	audio   []byte
	seek    []int64
	end     int
}

// frame returns the frame bytes aligned to the first frame's word boundary.
// skip is the bit index of the frame's first bit inside that window.
func (f *apeFile) frame(idx int) ([]byte, uint32) {
	start64 := f.seek[idx]
	if start64 < 0 || start64 > int64(len(f.audio)) {
		return nil, 0
	}

	start := int(start64)
	remainder := int(start64-f.seek[0]) & (wordSize - 1)
	aligned := start - remainder

	stop64 := int64(f.end)
	if idx+1 < len(f.seek) {
		stop64 = f.seek[idx+1]
	}

	stop64 += wordSize
	if stop64 > int64(len(f.audio)) {
		stop64 = int64(len(f.audio))
	}

	stop := int(stop64)
	if aligned < 0 || aligned > stop {
		return nil, 0
	}

	return f.audio[aligned:stop], uint32(remainder * 8)
}

//nolint:cyclop,funlen // container versions share one parser
func parseFile(raw []byte) (apeFile, error) {
	if len(raw) < 8 || (string(raw[:4]) != magicMAC && string(raw[:4]) != magicMACF) {
		return apeFile{}, ErrUnsupported
	}

	version := int(binary.LittleEndian.Uint16(raw[4:]))
	if version < version3980 {
		return parseOld(raw, version)
	}

	if len(raw) < descriptorSize+headerSize {
		return apeFile{}, errShortDescriptor
	}

	descBytes := int(binary.LittleEndian.Uint32(raw[8:]))
	headBytes := int(binary.LittleEndian.Uint32(raw[12:]))
	seekBytesN := int(binary.LittleEndian.Uint32(raw[16:]))
	headerData := int(binary.LittleEndian.Uint32(raw[20:]))
	frameBytes := int(binary.LittleEndian.Uint32(raw[24:])) + int(binary.LittleEndian.Uint32(raw[28:]))<<32

	if descBytes < descriptorSize || headBytes < headerSize || len(raw) < descBytes+headBytes+seekBytesN {
		return apeFile{}, errShortDescriptor
	}

	head := raw[descBytes : descBytes+headBytes]
	level := Compression(binary.LittleEndian.Uint16(head[0:]))
	flags := binary.LittleEndian.Uint16(head[2:])
	frameBlocks := int(binary.LittleEndian.Uint32(head[4:]))
	finalBlocks := int(binary.LittleEndian.Uint32(head[8:]))
	frames := int(binary.LittleEndian.Uint32(head[12:]))
	stream := Stream{
		Bits:       int(binary.LittleEndian.Uint16(head[16:])),
		Channels:   int(binary.LittleEndian.Uint16(head[18:])),
		SampleRate: int(binary.LittleEndian.Uint32(head[20:])),
		Float:      raw[3] == 'F' || flags&flagFloat != 0,
	}

	err := checkFrames(level, frameBlocks, finalBlocks)
	if err != nil || !stream.supported() {
		return apeFile{}, ErrUnsupported
	}

	if frames <= 0 || frames > len(raw)/wordSize || seekBytesN < frames*wordSize {
		return apeFile{}, errEmptyAudio
	}

	seekAt := descBytes + headBytes

	seek := widenSeek(raw[seekAt:], frames)

	audioStart := seekAt + seekBytesN

	audioEnd := min(audioStart+headerData+frameBytes, len(raw))

	samples, err := sampleCount(frames, frameBlocks, finalBlocks)
	if err != nil {
		return apeFile{}, err
	}

	_, err = pcmBytes(samples, stream.blockAlign())
	if err != nil {
		return apeFile{}, err
	}

	return apeFile{
		stream:  stream,
		level:   level,
		version: version,
		frames:  frames,
		samples: samples,
		audio:   raw,
		seek:    seek,
		end:     audioEnd,
		blocks: func(idx int) int {
			if idx == frames-1 {
				return finalBlocks
			}

			return frameBlocks
		},
	}, nil
}

//nolint:cyclop,funlen // silence, pseudo-stereo, stereo, and multi-channel are separate paths
func decodeFrame(
	frame []byte, skip uint32, blocks int, stream Stream, level Compression, version int, interim bool,
) ([]byte, error) {
	if len(frame) < 8 || blocks < 0 {
		return nil, errShortFrame
	}

	reader := newRangeReader(frame, skip)
	reader.version = version
	crc := reader.readBits(32)

	special := uint32(0)
	if crc&(1<<crcHighBit) != 0 {
		special = reader.readBits(32)
	}

	reader.startCoder()

	size, err := pcmBytes(blocks, stream.blockAlign())
	if err != nil {
		return nil, err
	}

	out := make([]byte, size)
	preds := make([]*decoder, stream.Channels)
	sums := make([]coderState, stream.Channels)

	for idx := range preds {
		preds[idx] = newDecoder(stream.Bits, level, version)
		preds[idx].useInterim(interim)
		sums[idx] = newCoderState()
	}

	var lastX int32

	ch := make([]int32, stream.Channels)

	for block := range blocks {
		clear(ch)

		switch {
		case stream.Channels == 1 && special&specialMonoSilence != 0:
		case stream.Channels == 2 && special&specialLeftSilence != 0 && special&specialRightSilence != 0:
		case stream.Channels == 2 && version < version3950 && special&specialPseudoStereo == 0:
			x := reader.decodeValue(&sums[0])
			y := reader.decodeValue(&sums[1])
			ch[0] = preds[0].decompress(x, 0)
			ch[1] = preds[1].decompress(y, 0)
		case stream.Channels == 2 && special&specialPseudoStereo != 0:
			ch[0] = preds[0].decompress(reader.decodeValue(&sums[0]), 0)
			lastX = ch[0]
		case stream.Channels == 2:
			y := reader.decodeValue(&sums[1])
			x := reader.decodeValue(&sums[0])
			ch[1] = preds[1].decompress(y, int64(lastX))
			ch[0] = preds[0].decompress(x, int64(ch[1]))
			lastX = ch[0]
		default:
			for idx := range ch {
				ch[idx] = preds[idx].decompress(reader.decodeValue(&sums[idx]), 0)
			}
		}

		writeBlock(out[block*stream.blockAlign():], ch, stream.Bits)
	}

	if reader.short {
		return nil, errShortFrame
	}

	if crc&^(1<<crcHighBit) != frameCRC(out, 0) {
		// 24-bit files written between MAC 5.01 and 8.51 keep the prediction
		// sum at 64 bits. The current path narrows each side first, so a
		// checksum miss is retried once in that interim mode.
		if stream.Bits == 24 && !interim {
			return decodeFrame(frame, skip, blocks, stream, level, version, true)
		}

		return nil, errFrameCRC
	}

	return out, nil
}

// widenSeek rebuilds absolute offsets. Stored seek entries are uint32 and wrap
// at 4 GiB; a smaller value than the previous entry starts the next 4 GiB.
func widenSeek(raw []byte, frames int) []int64 {
	seek := make([]int64, frames)

	var (
		add  int64
		prev uint32
	)

	for idx := range frames {
		cur := binary.LittleEndian.Uint32(raw[idx*wordSize:])
		if idx > 0 && cur < prev {
			add += 1 << 32
		}

		seek[idx] = add + int64(cur)
		prev = cur
	}

	return seek
}

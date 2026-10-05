package ape

import (
	"encoding/binary"
	"hash/crc32"
)

// Special-frame flags written after the frame CRC when a channel is silent
// or the stereo pair is identical. From Prepare.h.
const (
	specialLeftSilence  = 1
	specialRightSilence = 2
	specialPseudoStereo = 4
	specialMonoSilence  = 1
)

type prepared struct {
	ch      [][]int32
	special int
	crc     uint32
}

func prepare(pcm []byte, channels, bits int) prepared {
	width := bits / 8
	blocks := len(pcm) / (channels * width)
	ch := make([][]int32, channels)

	for idx := range ch {
		ch[idx] = make([]int32, blocks)
	}

	if channels <= 2 {
		return prepareFew(pcm, ch, bits, width)
	}

	if (bits == 16 || bits == 24) && (channels == 4 || channels >= 6) {
		preparePairs(pcm, ch, bits, width)
	} else {
		preparePlain(pcm, ch, bits, width)
	}

	return prepared{ch: ch, crc: frameCRC(pcm, 0)}
}

func prepareFew(pcm []byte, ch [][]int32, bits, width int) prepared {
	var lPeak, rPeak int

	for idx := range ch[0] {
		off := idx * len(ch) * width

		right := readSample(pcm[off:], bits)
		if len(ch) == 1 {
			rPeak = max(rPeak, absInt(int(right)))
			ch[0][idx] = right

			continue
		}

		left := readSample(pcm[off+width:], bits)
		lPeak = max(lPeak, absInt(int(left)))
		rPeak = max(rPeak, absInt(int(right)))
		ch[0][idx], ch[1][idx] = midSide(right, left)
	}

	special := 0

	if bits == 16 {
		var side []int32
		if len(ch) == 2 {
			side = ch[1]
		}

		special = specialCodes(len(ch), lPeak, rPeak, side)
	}

	return prepared{ch: ch, special: special, crc: frameCRC(pcm, special)}
}

func preparePlain(pcm []byte, ch [][]int32, bits, width int) {
	channels := len(ch)
	for idx := range ch[0] {
		off := idx * channels * width
		for c := range channels {
			ch[c][idx] = readSample(pcm[off+c*width:], bits)
		}
	}
}

func preparePairs(pcm []byte, ch [][]int32, bits, width int) {
	channels := len(ch)
	for idx := range ch[0] {
		off := idx * channels * width
		pos := 0
		next := func() int32 {
			sample := readSample(pcm[off+pos*width:], bits)
			pos++

			return sample
		}
		putPair := func(xCh, yCh int) {
			ch[xCh][idx], ch[yCh][idx] = midSide(next(), next())
		}

		putPair(0, 1)

		if channels == 4 {
			putPair(2, 3)

			continue
		}

		ch[2][idx] = next()
		ch[3][idx] = next()

		putPair(4, 5)

		if channels >= 8 {
			putPair(6, 7)
		}

		start := 8
		if channels == 7 {
			start = 6
		}

		for c := start; c < channels; c++ {
			ch[c][idx] = next()
		}
	}
}

func undoMidSide(x, y int32) (int32, int32) {
	right := x - y/2

	return right, right + y
}

func writeSample(dst []byte, sample int32, bits int) {
	switch bits {
	case 8:
		dst[0] = byte(sample + 128)
	case 24:
		dst[0] = byte(sample)
		dst[1] = byte(sample >> 8)
		dst[2] = byte(sample >> 16)
	case 32:
		binary.LittleEndian.PutUint32(dst, uint32(sample))
	default:
		binary.LittleEndian.PutUint16(dst, uint16(sample))
	}
}

func writeBlock(dst []byte, ch []int32, bits int) {
	width := bits / 8
	channels := len(ch)

	if channels <= 2 {
		if channels == 1 {
			writeSample(dst, ch[0], bits)

			return
		}

		right, left := undoMidSide(ch[0], ch[1])
		writeSample(dst, right, bits)
		writeSample(dst[width:], left, bits)

		return
	}

	if (bits == 16 || bits == 24) && (channels == 4 || channels >= 6) {
		writePairs(dst, ch, bits, width)

		return
	}

	for idx := range channels {
		writeSample(dst[idx*width:], ch[idx], bits)
	}
}

func writePairs(dst []byte, ch []int32, bits, width int) {
	pos := 0
	put := func(sample int32) {
		writeSample(dst[pos*width:], sample, bits)
		pos++
	}
	putPair := func(x, y int32) {
		right, left := undoMidSide(x, y)
		put(right)
		put(left)
	}

	putPair(ch[0], ch[1])

	if len(ch) == 4 {
		putPair(ch[2], ch[3])

		return
	}

	put(ch[2])
	put(ch[3])
	putPair(ch[4], ch[5])

	if len(ch) >= 8 {
		putPair(ch[6], ch[7])
	}

	start := 8
	if len(ch) == 7 {
		start = 6
	}

	for idx := start; idx < len(ch); idx++ {
		put(ch[idx])
	}
}

func midSide(right, left int32) (int32, int32) {
	diff := left - right

	return right + diff/2, diff
}

func specialCodes(channels, lPeak, rPeak int, side []int32) int {
	if channels == 1 {
		if rPeak == 0 {
			return specialMonoSilence
		}

		return 0
	}

	special := 0
	if lPeak == 0 {
		special |= specialLeftSilence
	}

	if rPeak == 0 {
		special |= specialRightSilence
	}

	if stereoIdentical(side) {
		special |= specialPseudoStereo
	}

	return special
}

func readSample(pcm []byte, bits int) int32 {
	switch bits {
	case 8:
		return int32(pcm[0]) - 128
	case 24:
		value := int32(pcm[0]) | int32(pcm[1])<<8 | int32(pcm[2])<<16

		return (value << 8) >> 8
	case 32:
		return int32(binary.LittleEndian.Uint32(pcm))
	default:
		return int32(int16(binary.LittleEndian.Uint16(pcm)))
	}
}

func stereoIdentical(samples []int32) bool {
	for i, sample := range samples {
		if sample != 0 {
			return false
		}

		if i+1 == len(samples) {
			return true
		}
	}

	return false
}

// frameCRC is the value stored ahead of the range coder: IEEE CRC of the
// raw PCM, shifted right one bit, with the high bit set when a special code follows.
func frameCRC(pcm []byte, special int) uint32 {
	sum := crc32.ChecksumIEEE(pcm) >> 1
	if special != 0 {
		sum |= 1 << crcHighBit
	}

	return sum
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}

	return v
}

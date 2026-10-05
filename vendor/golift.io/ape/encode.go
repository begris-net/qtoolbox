package ape

import (
	"encoding/binary"
	"fmt"
	"hash"
	"io"
)

// Encode writes an APE file of interleaved little-endian PCM.
// dst must be seekable. Writing starts at offset zero, and the descriptor,
// header, and seek table are patched after the frames are written.
// A destination longer than the new file is not truncated.
func Encode(dst io.WriteSeeker, pcm []byte, stream Stream, opt *Options) error {
	err := validate(pcm, stream)
	if err != nil {
		return err
	}

	enc, err := NewEncoder(dst, stream, len(pcm)/stream.blockAlign(), opt)
	if err != nil {
		return err
	}

	err = enc.Write(pcm)
	if err != nil {
		return err
	}

	return enc.Close()
}

func closeFile(
	dst io.WriteSeeker,
	sum hash.Hash,
	seek []uint32,
	stream Stream,
	level Compression,
	opt *Options,
	frameBlocks, samples, frames, prefix int,
) error {
	footer := opt.footer()

	err := writeExtra(dst, sum, footer)
	if err != nil {
		return err
	}

	tail, err := dst.Seek(0, io.SeekCurrent)
	if err != nil {
		return fmt.Errorf("ape: telling file size: %w", err)
	}

	frameBytes := int(tail) - prefix - len(opt.header()) - len(footer)

	err = finishFile(dst, sum, seek, stream, level, opt, frameBlocks, samples, frames, frameBytes)
	if err != nil {
		return err
	}

	_, err = dst.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("ape: seeking tag: %w", err)
	}

	return writeTag(dst, opt.tags())
}

func (o *Options) header() []byte {
	if o == nil {
		return nil
	}

	return o.Header
}

func (o *Options) footer() []byte {
	if o == nil {
		return nil
	}

	return o.Footer
}

func (o *Options) tags() map[string]string {
	if o == nil {
		return nil
	}

	return o.Tags
}

func writeExtra(dst io.Writer, sum hash.Hash, data []byte) error {
	if len(data) == 0 {
		return nil
	}

	sum.Write(data)

	_, err := dst.Write(data)
	if err != nil {
		return fmt.Errorf("ape: writing container data: %w", err)
	}

	return nil
}

// stitchFrame joins the unwritten tail of the previous frame onto this one.
// It follows APECompressCreate::FixupFrame: swap each word, slide the new
// frame forward, copy the previous word's bytes into the hole, and swap back.
func stitchFrame(payload []byte, prev uint32, prevLen int) ([]byte, uint32, int) {
	nBytes := len(payload)
	if prevLen == 0 {
		return payload, wordAt(payload, nBytes/wordSize*wordSize), nBytes % wordSize
	}

	nWords := nBytes/wordSize + 1
	buf := make([]byte, nBytes+prevLen+wordSize)
	copy(buf, payload)

	swapWords(buf, nWords)
	copy(buf[prevLen:], buf[:nBytes])

	var prevBytes [4]byte
	binary.LittleEndian.PutUint32(prevBytes[:], prev)
	copy(buf[:prevLen], prevBytes[:prevLen])
	swapWords(buf, nWords)

	total := nBytes + prevLen

	return buf[:total], wordAt(buf, total/wordSize*wordSize), total % wordSize
}

func wordAt(buf []byte, off int) uint32 {
	var word [4]byte
	if off < len(buf) {
		copy(word[:], buf[off:])
	}

	return binary.LittleEndian.Uint32(word[:])
}

func swapWords(buf []byte, words int) {
	for i := range words {
		off := i * wordSize
		if off+wordSize > len(buf) {
			return
		}

		buf[off], buf[off+3] = buf[off+3], buf[off]
		buf[off+1], buf[off+2] = buf[off+2], buf[off+1]
	}
}

func finishFile(
	dst io.WriteSeeker,
	sum hash.Hash,
	seek []uint32,
	stream Stream,
	level Compression,
	opt *Options,
	frameBlocks, samples, frames, frameBytes int,
) error {
	finalBlocks := samples % frameBlocks
	if finalBlocks == 0 {
		finalBlocks = frameBlocks
	}

	header := headerBytes(stream, level, opt, frameBlocks, finalBlocks, frames)
	table := seekBytes(seek)

	sum.Write(header)
	sum.Write(table)

	desc := descriptorBytes(stream, opt, frames, frameBytes, sum.Sum(nil))

	_, err := dst.Seek(0, io.SeekStart)
	if err != nil {
		return fmt.Errorf("ape: seeking descriptor: %w", err)
	}

	_, err = dst.Write(desc)
	if err != nil {
		return fmt.Errorf("ape: writing descriptor: %w", err)
	}

	_, err = dst.Write(header)
	if err != nil {
		return fmt.Errorf("ape: writing header: %w", err)
	}

	_, err = dst.Write(table)
	if err != nil {
		return fmt.Errorf("ape: writing seek table: %w", err)
	}

	return nil
}

func descriptorBytes(stream Stream, opt *Options, frames, frameBytes int, sum []byte) []byte {
	buf := make([]byte, descriptorSize)
	copy(buf, "MAC ")

	if stream.Float {
		buf[3] = 'F'
	}

	binary.LittleEndian.PutUint16(buf[4:], fileVersion)
	binary.LittleEndian.PutUint16(buf[6:], programVersion)
	binary.LittleEndian.PutUint32(buf[8:], descriptorSize)
	binary.LittleEndian.PutUint32(buf[12:], headerSize)
	binary.LittleEndian.PutUint32(buf[16:], uint32(frames*wordSize))
	binary.LittleEndian.PutUint32(buf[20:], uint32(len(opt.header())))
	binary.LittleEndian.PutUint32(buf[24:], uint32(frameBytes))
	binary.LittleEndian.PutUint32(buf[28:], uint32(frameBytes>>32)) //nolint:mnd // high half of the byte count
	binary.LittleEndian.PutUint32(buf[32:], uint32(len(opt.footer())))
	copy(buf[36:], sum)

	return buf
}

func headerBytes(stream Stream, level Compression, opt *Options, frameBlocks, finalBlocks, frames int) []byte {
	flags := uint16(flagCreateWAV)
	if len(opt.header()) > 0 {
		flags = uint16(opt.Format)
	}

	if stream.Float {
		flags |= flagFloat
	}

	buf := make([]byte, headerSize)
	binary.LittleEndian.PutUint16(buf[0:], uint16(level))
	binary.LittleEndian.PutUint16(buf[2:], flags)
	binary.LittleEndian.PutUint32(buf[4:], uint32(frameBlocks))
	binary.LittleEndian.PutUint32(buf[8:], uint32(finalBlocks))
	binary.LittleEndian.PutUint32(buf[12:], uint32(frames))
	binary.LittleEndian.PutUint16(buf[16:], uint16(stream.Bits))
	binary.LittleEndian.PutUint16(buf[18:], uint16(stream.Channels))
	binary.LittleEndian.PutUint32(buf[20:], uint32(stream.SampleRate))

	return buf
}

func seekBytes(seek []uint32) []byte {
	buf := make([]byte, len(seek)*wordSize)
	for i, off := range seek {
		binary.LittleEndian.PutUint32(buf[i*wordSize:], off)
	}

	return buf
}

func writeZeros(dst io.Writer, n int) error {
	_, err := dst.Write(make([]byte, n))
	if err != nil {
		return fmt.Errorf("ape: writing header placeholder: %w", err)
	}

	return nil
}

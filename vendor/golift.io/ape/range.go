package ape

import "encoding/binary"

// Range coder from BitArray.cpp. The probability tables are the fast model's
// published overflow distribution; a decoder has to use the same numbers.
const (
	codeBits    = 32
	topValue    = uint32(1) << (codeBits - 1)
	shiftBits   = codeBits - 9
	bottomValue = topValue >> 8

	rangeOverflowShift = 16
	modelElements      = 64
	overflowPivot      = 32768
	overflowSignal     = 1
	kSumStart          = (1 << 10) * 16
)

type bitArray struct {
	words []uint32
	bit   uint32
	low   uint32
	rang  uint32
	help  uint32
	buf   byte
}

func newBitArray() *bitArray {
	return &bitArray{words: make([]uint32, 1024)}
}

func (b *bitArray) reset() {
	clear(b.words)
	b.bit = 0
	b.low = 0
	b.rang = 0
	b.help = 0
	b.buf = 0
}

func (b *bitArray) ensure(extra uint32) {
	need := int((b.bit + extra + 31) >> 5)
	if need < len(b.words) {
		return
	}

	next := make([]uint32, need+need/5+8)
	copy(next, b.words)
	b.words = next
}

func (b *bitArray) putc(v byte) {
	b.ensure(8)
	b.words[b.bit>>5] |= uint32(v) << (24 - (b.bit & 31))
	b.bit += 8
}

func (b *bitArray) encodeU32(n uint32) {
	b.ensure(64)

	index := b.bit >> 5
	bitIndex := b.bit & 31

	if bitIndex == 0 {
		b.words[index] = n
	} else {
		b.words[index] |= n >> bitIndex
		b.words[index+1] = n << (32 - bitIndex)
	}

	b.bit += 32
}

func (b *bitArray) normalize() {
	for b.rang <= bottomValue {
		switch {
		case b.low < (0xFF << shiftBits):
			b.putc(b.buf)

			for ; b.help > 0; b.help-- {
				b.putc(0xFF)
			}

			b.buf = byte(b.low >> shiftBits)
		case b.low&topValue != 0:
			b.putc(b.buf + 1)
			b.bit += b.help * 8
			b.ensure(0)
			b.help = 0
			b.buf = byte(b.low >> shiftBits)
		default:
			b.help++
		}

		b.low = (b.low << 8) & (topValue - 1)
		b.rang <<= 8
	}
}

func (b *bitArray) encodeFast(width, total uint32) {
	b.normalize()

	temp := b.rang >> rangeOverflowShift
	b.rang = temp * width
	b.low += temp * total
}

func (b *bitArray) encodeDirect(value uint32) {
	b.normalize()
	b.rang >>= 16
	b.low += b.rang * value
}

func (b *bitArray) encodeValue(n int64, kSum *uint32) {
	if n > 0 {
		n = n*2 - 1
	} else {
		n = -n * 2
	}

	pivot := max(*kSum/32, 1)

	overflow64 := uint64(n) / uint64(pivot)
	overflow := uint32(overflow64)

	if uint64(overflow) != overflow64 {
		pivot = overflowPivot
		overflow = uint32(uint64(n) / uint64(pivot))

		b.encodeFast(rangeWidth[modelElements-1], rangeTotal[modelElements-1])
		b.encodeDirect((overflowSignal >> 16) & 0xFFFF)
		b.encodeDirect(overflowSignal & 0xFFFF)
	}

	base := uint32(n - int64(overflow)*int64(pivot))
	*kSum += uint32((n+1)/2) - ((*kSum + 16) >> 5)

	if overflow < modelElements-1 {
		b.encodeFast(rangeWidth[overflow], rangeTotal[overflow])
	} else {
		b.encodeFast(rangeWidth[modelElements-1], rangeTotal[modelElements-1])
		b.encodeDirect((overflow >> 16) & 0xFFFF)
		b.encodeDirect(overflow & 0xFFFF)
	}

	b.encodeBase(base, pivot)
}

func (b *bitArray) encodeBase(base, pivot uint32) {
	if pivot < 1<<16 {
		b.normalize()

		temp := b.rang / pivot
		b.rang = temp
		b.low += temp * base

		return
	}

	bits := uint32(0)
	for pivot>>bits > 0 {
		bits++
	}

	shift := uint32(0)
	if bits >= 16 {
		shift = bits - 16
	}

	split := uint32(1) << shift
	b.encodePivotPiece(base/split, pivot/split+1)
	b.encodePivotPiece(base%split, split)
}

func (b *bitArray) encodePivotPiece(base, pivot uint32) {
	b.normalize()

	temp := b.rang / pivot
	b.rang = temp
	b.low += temp * base
}

func (b *bitArray) flushCoder() {
	for b.bit%8 != 0 {
		b.bit++
	}

	b.low = 0
	b.rang = topValue
	b.buf = 0
	b.help = 0
}

func (b *bitArray) finalize() {
	b.normalize()

	temp := (b.low >> shiftBits) + 1
	if temp > 0xFF {
		b.putc(b.buf + 1)

		for ; b.help > 0; b.help-- {
			b.putc(0)
		}
	} else {
		b.putc(b.buf)

		for ; b.help > 0; b.help-- {
			b.putc(0xFF)
		}
	}

	b.putc(byte(temp & 0xFF))
	b.putc(0)
	b.putc(0)
	b.putc(0)
}

const extraBits = (codeBits-2)%8 + 1

type rangeReader struct {
	words   []uint32
	limit   int
	bit     uint32
	low     uint32
	rang    uint32
	buf     uint32
	version int
	short   bool
}

// coderState is the per-channel rice state. k is used by versions before 3990.
type coderState struct {
	k   uint32
	sum uint32
}

func newCoderState() coderState {
	return coderState{k: 10, sum: kSumStart}
}

func newRangeReader(frame []byte, bit uint32) *rangeReader {
	// The extra words keep a lookahead index in range. have uses limit, so
	// those zeros are not treated as frame data.
	words := make([]uint32, len(frame)/wordSize+4)
	limit := 0

	for i := 0; i+wordSize <= len(frame); i += wordSize {
		words[limit] = binary.LittleEndian.Uint32(frame[i:])
		limit++
	}

	if rem := len(frame) % wordSize; rem != 0 {
		var tail [wordSize]byte
		copy(tail[:], frame[len(frame)-rem:])
		words[limit] = binary.LittleEndian.Uint32(tail[:])
		limit++
	}

	return &rangeReader{words: words, limit: limit, bit: bit}
}

func (r *rangeReader) readBits(n uint32) uint32 {
	left := 32 - (r.bit & 31)
	index := r.bit >> 5
	r.bit += n

	if !r.have(index) || (left < n && !r.have(index+1)) {
		return 0
	}

	if left >= n {
		return (r.words[index] & ((1 << left) - 1)) >> (left - n)
	}

	right := n - left
	high := (r.words[index] & ((1 << left) - 1)) << right
	low := r.words[index+1] >> (32 - right)

	return high | low
}

func (r *rangeReader) readByte() uint32 {
	index := r.bit >> 5
	if !r.have(index) {
		r.bit += 8

		return 0
	}

	value := (r.words[index] >> (24 - (r.bit & 31))) & 0xFF
	r.bit += 8

	return value
}

func (r *rangeReader) have(index uint32) bool {
	if int(index) < r.limit {
		return true
	}

	r.short = true

	return false
}

func (r *rangeReader) startCoder() {
	for r.bit%8 != 0 {
		r.bit++
	}

	_ = r.readBits(8)
	r.buf = r.readBits(8)
	r.low = r.buf >> (8 - extraBits)
	r.rang = 1 << extraBits
}

func (r *rangeReader) fill() {
	for r.rang <= bottomValue {
		if r.rang == 0 {
			return
		}

		r.bufShift()
	}
}

func (r *rangeReader) bufShift() {
	r.buf = (r.buf << 8) | r.readByte()
	r.low = (r.low << 8) | ((r.buf >> 1) & 0xFF)
	r.rang <<= 8
}

func (r *rangeReader) decodeFast(shift uint32) uint32 {
	r.fill()
	r.rang >>= shift

	if r.rang == 0 {
		return 0
	}

	return r.low / r.rang
}

func (r *rangeReader) decodeFastUpdate(shift uint32) uint32 {
	r.fill()
	r.rang >>= shift

	if r.rang == 0 {
		return 0
	}

	result := r.low / r.rang
	r.low %= r.rang

	return result
}

func (r *rangeReader) overflow(pivot *uint32) uint32 {
	for {
		if r.short {
			return 0
		}

		total := r.decodeFast(rangeOverflowShift)
		symbol := uint32(0)

		for symbol < modelElements-1 && total >= rangeTotal[symbol]+rangeWidth[symbol] {
			symbol++
		}

		r.low -= r.rang * rangeTotal[symbol]
		r.rang *= rangeWidth[symbol]

		if symbol != modelElements-1 {
			return symbol
		}

		symbol = r.decodeFastUpdate(16) << 16
		symbol |= r.decodeFastUpdate(16)

		if symbol != overflowSignal || r.short {
			return symbol
		}

		*pivot = overflowPivot
	}
}

func (r *rangeReader) decodeValue(st *coderState) int64 {
	if r.version < fileVersion {
		return r.decodeLegacy(st)
	}

	kSum := &st.sum
	pivot := max(*kSum/32, 1)
	over := r.overflow(&pivot)

	var base uint32

	if pivot >= 1<<16 {
		bits := uint32(0)
		for pivot>>bits > 0 {
			bits++
		}

		shift := uint32(0)
		if bits >= 16 {
			shift = bits - 16
		}

		split := uint32(1) << shift
		base = r.pivotPiece(pivot/split+1)*split + r.pivotPiece(split)
	} else {
		base = r.pivotPiece(pivot)
	}

	value := int64(base) + int64(over)*int64(pivot)
	*kSum += uint32((value+1)/2) - ((*kSum + 16) >> 5)

	if value&1 != 0 {
		return (value >> 1) + 1
	}

	return -(value >> 1)
}

//nolint:cyclop // the pre-3990 range coder branches on overflow and k
func (r *rangeReader) decodeLegacy(st *coderState) int64 {
	total := r.decodeFast(rangeOverflowShift)

	symbol := uint32(0)
	for symbol < modelElements-1 && total >= rangeTotal1[symbol]+rangeWidth1[symbol] {
		symbol++
	}

	r.low -= r.rang * rangeTotal1[symbol]
	r.rang *= rangeWidth1[symbol]

	tempK := uint32(0)
	if symbol == modelElements-1 {
		tempK = r.decodeFastUpdate(5)
		symbol = 0
	} else if st.k >= 1 {
		tempK = st.k - 1
	}

	var value int64
	if tempK <= 16 || r.version < 3910 {
		value = int64(r.decodeFastUpdate(tempK))
	} else {
		x1 := r.decodeFastUpdate(16)
		x2 := r.decodeFastUpdate(tempK - 16)
		value = int64(x1) | (int64(x2) << 16)
	}

	value += int64(symbol) << tempK
	st.sum += uint32((value+1)/2) - ((st.sum + 16) >> 5)

	if st.k < 31 && st.sum < kSumMin[st.k] {
		st.k--
	} else if st.k < 31 && kSumMin[st.k+1] != 0 && st.sum >= kSumMin[st.k+1] {
		st.k++
	}

	if value&1 != 0 {
		return (value >> 1) + 1
	}

	return -(value >> 1)
}

func (r *rangeReader) pivotPiece(pivot uint32) uint32 {
	r.fill()

	if r.rang == 0 || pivot == 0 {
		return 0
	}

	r.rang /= pivot
	if r.rang == 0 {
		return 0
	}

	base := r.low / r.rang
	r.low %= r.rang

	return base
}

func (b *bitArray) payload() []byte {
	n := int(b.bit / 8)
	out := make([]byte, n)

	for i := range n {
		word := b.words[i/4]
		out[i] = byte(word >> ((i % 4) * 8))
	}

	return out
}

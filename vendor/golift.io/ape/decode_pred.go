package ape

import "slices"

// decoder is the inverse of predictor, from CPredictorDecompress3950toCurrent.
type decoder struct {
	predA   roll
	predB   roll
	adaptA  roll
	adaptB  roll
	index   int
	bits    int
	lastA   int64
	filterA int32
	filterB int32
	coeffA  [8]int64
	coeffB  [8]int64
	nn      []*nnFilter
	legacy  bool
	interim bool
}

func (d *decoder) useInterim(on bool) {
	if !on {
		return
	}

	d.interim = true

	for _, filter := range d.nn {
		filter.interim = true
	}
}

func newDecoder(bits int, level Compression, version int) *decoder {
	dec := &decoder{
		predA:  newRoll(windowBlocks, 8),
		predB:  newRoll(windowBlocks, 8),
		adaptA: newRoll(windowBlocks, 8),
		adaptB: newRoll(windowBlocks, 8),
		bits:   bits,
		legacy: version < version3950,
	}
	for _, stage := range level.stages() {
		filter := newNNFilter(stage.order, stage.shift, bits)
		filter.oldDelta = version < version3980
		dec.nn = append(dec.nn, filter)
	}

	dec.flush()

	return dec
}

func (d *decoder) flush() {
	d.predA.flush()
	d.predB.flush()
	d.adaptA.flush()
	d.adaptB.flush()
	d.lastA = 0
	d.filterA = 0
	d.filterB = 0
	d.index = 0
	d.coeffA = [8]int64{}
	d.coeffB = [8]int64{}
	d.coeffA[0] = coeff0
	d.coeffA[1] = coeff1
	d.coeffA[2] = coeff2
	d.coeffA[3] = coeff3

	for _, filter := range d.nn {
		filter.flush()
	}
}

func (d *decoder) decompress3930(sample int64) int32 {
	if d.index == windowBlocks {
		d.predA.roll()
		d.index = 0
	}

	input := int32(sample)
	for _, filter := range slices.Backward(d.nn) {
		input = int32(filter.decompress(int64(input)))
	}

	p1 := int32(d.predA.get(-1))
	p2 := p1 - int32(d.predA.get(-2))
	p3 := int32(d.predA.get(-2)) - int32(d.predA.get(-3))
	p4 := int32(d.predA.get(-3)) - int32(d.predA.get(-4))
	sum := int32(mul32(int64(p1), d.coeffA[0]) + mul32(int64(p2), d.coeffA[1]) +
		mul32(int64(p3), d.coeffA[2]) + mul32(int64(p4), d.coeffA[3]))
	stored := input + (sum >> 9)
	d.predA.set(0, int64(stored))

	direction := boolToInt(input < 0) - boolToInt(input > 0)
	d.coeffA[0] = int64(int32(d.coeffA[0]) + adapt3930(p1)*direction)
	d.coeffA[1] = int64(int32(d.coeffA[1]) + adapt3930(p2)*direction)
	d.coeffA[2] = int64(int32(d.coeffA[2]) + adapt3930(p3)*direction)
	d.coeffA[3] = int64(int32(d.coeffA[3]) + adapt3930(p4)*direction)

	result := stored + ((int32(d.lastA) * 31) >> 5)
	d.lastA = int64(result)
	d.predA.inc()
	d.index++

	return result
}

func adapt3930(value int32) int32 {
	return ((value >> 30) & 2) - 1
}

func (d *decoder) decompress(sampleA, sampleB int64) int32 {
	if d.legacy {
		return d.decompress3930(sampleA)
	}

	if d.index == windowBlocks {
		d.predA.roll()
		d.predB.roll()
		d.adaptA.roll()
		d.adaptB.roll()
		d.index = 0
	}

	for _, v := range slices.Backward(d.nn) {
		sampleA = v.decompress(sampleA)
	}

	d.predA.set(0, d.lastA)
	d.predA.set(-1, d.wrap(d.predA.get(0)-d.predA.get(-1)))
	d.predB.set(0, stageCompress(&d.filterB, int32(sampleB), d.bits >= 32))
	d.predB.set(-1, d.wrap(d.predB.get(0)-d.predB.get(-1)))

	predA, predB := d.predictions()
	current := d.combine(sampleA, predA, predB)

	d.adaptA.set(0, adaptSign(d.predA.get(0)))
	d.adaptA.set(-1, adaptSign(d.predA.get(-1)))
	d.adaptB.set(0, adaptSign(d.predB.get(0)))
	d.adaptB.set(-1, adaptSign(d.predB.get(-1)))

	direction := int64(boolToInt(sampleA < 0) - boolToInt(sampleA > 0))
	for idx := range 4 {
		d.coeffA[idx] = d.wrap(d.coeffA[idx] + d.adaptA.get(-idx)*direction)
	}

	for idx := range 5 {
		d.coeffB[idx] = d.wrap(d.coeffB[idx] + d.adaptB.get(-idx)*direction)
	}

	result := stageDecompress(&d.filterA, current)
	d.lastA = current
	d.predA.inc()
	d.predB.inc()
	d.adaptA.inc()
	d.adaptB.inc()
	d.index++

	return result
}

func (d *decoder) predictions() (int64, int64) {
	if d.bits <= 16 {
		return d.dot32(true), d.dot32(false)
	}

	return d.dot64(true), d.dot64(false)
}

func (d *decoder) dot32(sideA bool) int64 {
	var sum int32

	if sideA {
		for idx := range 4 {
			sum += int32(d.predA.get(-idx)) * int32(d.coeffA[idx])
		}

		return int64(sum)
	}

	for idx := range 5 {
		sum += int32(d.predB.get(-idx)) * int32(d.coeffB[idx])
	}

	return int64(sum)
}

func (d *decoder) dot64(sideA bool) int64 {
	if sideA {
		var sum int64
		for idx := range 4 {
			sum += d.predA.get(-idx) * d.coeffA[idx]
		}

		return sum
	}

	var sum int64
	for idx := range 5 {
		sum += d.predB.get(-idx) * d.coeffB[idx]
	}

	return sum
}

func stageCompress(last *int32, input int32, wide bool) int64 {
	result := int64(input) - ((int64(*last) * stageMultiply) >> stageShift)
	if !wide {
		result = int64(int32(result))
	}

	*last = input

	return result
}

func stageDecompress(last *int32, input int64) int32 {
	*last = int32(input + ((int64(*last) * stageMultiply) >> stageShift))

	return *last
}

func (d *decoder) combine(sampleA, predA, predB int64) int64 {
	if d.bits >= 32 {
		return sampleA + ((predA + (predB >> 1)) >> predShift)
	}

	if d.interim {
		shifted := (predA + (predB >> 1)) >> predShift

		return d.wrap(sampleA + int64(int32(shifted)))
	}

	combined := (int32(predA) + (int32(predB) >> 1)) >> predShift

	return d.wrap(sampleA + int64(combined))
}

func (d *decoder) wrap(v int64) int64 {
	if d.bits < 32 {
		return int64(int32(v))
	}

	return v
}

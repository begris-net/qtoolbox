package ape

// Predictor coefficients and the first-order filter, from NewPredictor.cpp
// and ScaledFirstOrderFilter.h. Fast compression stops after this stage.
// Normal and above run neural-network filters on the residual, in this order:
//
//	Normal:      order 16, shift 11
//	High:        order 64, shift 11
//	Extra high:  order 256 shift 13, then order 32 shift 10
//	Insane:      order 1280 shift 15, then order 256 shift 13, then order 16 shift 11
const (
	windowBlocks = 256
	predHistory  = 10
	adaptHistory = 9
	coeffCount   = 9

	stageMultiply = 31
	stageShift    = 5
	predShift     = 10

	coeff0 = 360
	coeff1 = 317
	coeff2 = -109
	coeff3 = 98

	coeffOrigin = 8
	signMask    = 2
	signShift   = 30
)

type predictor struct {
	pred  roll
	adapt roll
	index int
	bits  int
	lastA int32
	lastB int32
	coeff [coeffCount]int64
	nn    []*nnFilter
}

func newPredictor(bits int, level Compression) *predictor {
	pred := &predictor{
		pred:  newRoll(windowBlocks, predHistory),
		adapt: newRoll(windowBlocks, adaptHistory),
		bits:  bits,
	}
	for _, stage := range level.stages() {
		pred.nn = append(pred.nn, newNNFilter(stage.order, stage.shift, bits))
	}

	pred.flush()

	return pred
}

func (p *predictor) flush() {
	p.pred.flush()
	p.adapt.flush()
	p.lastA = 0
	p.lastB = 0
	p.index = 0
	p.coeff = [coeffCount]int64{}
	p.coeff[8] = coeff0
	p.coeff[7] = coeff1
	p.coeff[6] = coeff2
	p.coeff[5] = coeff3

	for _, filter := range p.nn {
		filter.flush()
	}
}

// compress is one sample. a is this channel, b is the other channel's
// prepared sample (previous X when compressing Y, current Y when compressing X).
func (p *predictor) compress(sampleA, sampleB int32) int64 {
	if p.index == windowBlocks {
		p.pred.roll()
		p.adapt.roll()
		p.index = 0
	}

	filtA := p.stage(&p.lastA, sampleA)
	filtB := p.stage(&p.lastB, sampleB)
	p.store(0, filtA)
	p.store(-2, p.pred.get(-1)-p.pred.get(-2))
	p.store(-5, filtB)
	p.store(-6, p.pred.get(-5)-p.pred.get(-6))

	output := p.residual(filtA)

	p.adapt.set(0, adaptSign(p.pred.get(-1)))
	p.adapt.set(-1, adaptSign(p.pred.get(-2)))
	p.adapt.set(-4, adaptSign(p.pred.get(-5)))
	p.adapt.set(-5, adaptSign(p.pred.get(-6)))

	direction := int64(boolToInt(output < 0) - boolToInt(output > 0))
	for idx := range coeffCount {
		next := p.coeff[idx] + p.adapt.get(idx-coeffOrigin)*direction
		if p.bits < 32 {
			next = int64(int32(next))
		}

		p.coeff[idx] = next
	}

	p.pred.inc()
	p.adapt.inc()
	p.index++

	for _, filter := range p.nn {
		output = filter.compress(output)
	}

	return output
}

func (p *predictor) stage(last *int32, input int32) int64 {
	result := int64(input) - ((int64(*last) * stageMultiply) >> stageShift)
	*last = input

	if p.bits < 32 {
		result = int64(int32(result))
	}

	return result
}

func (p *predictor) store(i int, v int64) {
	if p.bits < 32 {
		v = int64(int32(v))
	}

	p.pred.set(i, v)
}

func (p *predictor) residual(filtA int64) int64 {
	predA, predB := p.predictions()
	if p.bits >= 32 {
		return filtA - ((predA + (predB >> 1)) >> predShift)
	}

	combined := (int32(predA) + (int32(predB) >> 1)) >> predShift

	return filtA - int64(combined)
}

func (p *predictor) predictions() (int64, int64) {
	if p.bits <= 16 {
		return p.dot32(true), p.dot32(false)
	}

	return p.dot64(true), p.dot64(false)
}

func (p *predictor) dot32(sideA bool) int64 {
	if sideA {
		return mul32(p.pred.get(-1), p.coeff[coeffOrigin]) +
			mul32(p.pred.get(-2), p.coeff[coeffOrigin-1]) +
			mul32(p.pred.get(-3), p.coeff[coeffOrigin-2]) +
			mul32(p.pred.get(-4), p.coeff[coeffOrigin-3])
	}

	return mul32(p.pred.get(-5), p.coeff[coeffOrigin-4]) +
		mul32(p.pred.get(-6), p.coeff[coeffOrigin-5]) +
		mul32(p.pred.get(-7), p.coeff[coeffOrigin-6]) +
		mul32(p.pred.get(-8), p.coeff[coeffOrigin-7]) +
		mul32(p.pred.get(-9), p.coeff[0])
}

func (p *predictor) dot64(sideA bool) int64 {
	if sideA {
		return p.pred.get(-1)*p.coeff[coeffOrigin] +
			p.pred.get(-2)*p.coeff[coeffOrigin-1] +
			p.pred.get(-3)*p.coeff[coeffOrigin-2] +
			p.pred.get(-4)*p.coeff[coeffOrigin-3]
	}

	return p.pred.get(-5)*p.coeff[coeffOrigin-4] +
		p.pred.get(-6)*p.coeff[coeffOrigin-5] +
		p.pred.get(-7)*p.coeff[coeffOrigin-6] +
		p.pred.get(-8)*p.coeff[coeffOrigin-7] +
		p.pred.get(-9)*p.coeff[0]
}

func mul32(left, right int64) int64 {
	return int64(int32(left) * int32(right))
}

func adaptSign(v int64) int64 {
	if v == 0 {
		return 0
	}

	return ((v >> signShift) & signMask) - 1
}

func boolToInt(v bool) int32 {
	if v {
		return 1
	}

	return 0
}

type roll struct {
	data []int64
	cur  int
	hist int
}

func newRoll(window, hist int) roll {
	return roll{data: make([]int64, window+hist), cur: hist, hist: hist}
}

func (r *roll) flush() {
	for i := 0; i <= r.hist; i++ {
		r.data[i] = 0
	}

	r.cur = r.hist
}

func (r *roll) roll() {
	copy(r.data[:r.hist], r.data[r.cur-r.hist:r.cur])
	r.cur = r.hist
}

func (r *roll) get(i int) int64 { return r.data[r.cur+i] }

func (r *roll) set(i int, v int64) { r.data[r.cur+i] = v }

func (r *roll) inc() { r.cur++ }

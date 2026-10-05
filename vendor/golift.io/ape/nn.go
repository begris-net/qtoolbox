package ape

// Neural-network filter from NNFilterGeneric.cpp. Normal compression is
// order 16 and shift 11. History is saturated to 16 bits. Coefficients
// are 16-bit below 32-bit samples and 32-bit at 32-bit samples.
const (
	nnWindow      = 512
	nnOrderNormal = 16
	nnShiftNormal = 11

	deltaLarge = 32
	deltaMid   = 16
	deltaSmall = 8
)

type nnFilter struct {
	order    int
	shift    uint
	round    int64
	wide     bool
	oldDelta bool
	interim  bool
	coeff    []int64
	input    roll
	delta    roll
	average  int64
	index    int
}

func newNNFilter(order, shift, bits int) *nnFilter {
	return &nnFilter{
		order: order,
		shift: uint(shift),
		round: 1 << (shift - 1),
		wide:  bits >= 32,
		coeff: make([]int64, order),
		input: newRoll(nnWindow, order),
		delta: newRoll(nnWindow, order),
	}
}

func (n *nnFilter) flush() {
	for i := range n.coeff {
		n.coeff[i] = 0
	}

	n.input.flush()
	n.delta.flush()
	n.average = 0
	n.index = 0
}

func (n *nnFilter) compress(input int64) int64 {
	if n.index == nnWindow {
		n.input.roll()
		n.delta.roll()
		n.index = 0
	}

	dot := n.dot()

	var output int64
	if n.wide {
		output = input - ((dot + n.round) >> n.shift)
	} else {
		shifted := (int32(dot) + int32(n.round)) >> n.shift
		output = int64(int32(input) - shifted)
	}

	n.adapt(output)
	n.updateDelta(input)
	n.input.set(0, int64(saturateInt16(input)))
	n.input.inc()
	n.delta.inc()
	n.index++

	return output
}

func (n *nnFilter) decompress(input int64) int64 {
	if n.index == nnWindow {
		n.input.roll()
		n.delta.roll()
		n.index = 0
	}

	dot := n.dot()

	var output int64

	switch {
	case n.wide:
		output = input + ((dot + n.round) >> n.shift)
	case n.interim:
		shifted := int32((dot + n.round) >> n.shift)
		output = int64(int32(input) + shifted)
	default:
		shifted := (int32(dot) + int32(n.round)) >> n.shift
		output = int64(int32(input) + shifted)
	}

	n.adapt(input)

	if n.oldDelta {
		n.updateDeltaOld(output)
	} else {
		n.updateDelta(output)
	}

	n.input.set(0, int64(saturateInt16(output)))
	n.input.inc()
	n.delta.inc()
	n.index++

	return output
}

func (n *nnFilter) dot() int64 {
	if n.wide {
		var sum int64

		for i := range n.order {
			product := int32(n.input.get(i-n.order)) * int32(n.coeff[i])
			sum += int64(product)
		}

		return sum
	}

	var sum int32
	for i := range n.order {
		sum += int32(n.input.get(i-n.order)) * int32(n.coeff[i])
	}

	return int64(sum)
}

func (n *nnFilter) adapt(output int64) {
	if output == 0 {
		return
	}

	for i := range n.order {
		next := int32(n.coeff[i])

		step := int32(n.delta.get(i - n.order))
		if output < 0 {
			next += step
		} else {
			next -= step
		}

		if n.wide {
			n.coeff[i] = int64(next)
		} else {
			n.coeff[i] = int64(int16(next))
		}
	}
}

func (n *nnFilter) updateDeltaOld(value int64) {
	delta := int64(0)

	sample := int32(value)
	if n.wide {
		if value != 0 {
			delta = ((value >> 28) & 8) - 4
		}
	} else if sample != 0 {
		delta = int64(((sample >> 28) & 8) - 4)
	}

	n.delta.set(0, delta)
	n.delta.set(-4, n.delta.get(-4)>>1)
	n.delta.set(-8, n.delta.get(-8)>>1)
}

func (n *nnFilter) updateDelta(input int64) {
	abs := input
	if abs < 0 {
		abs = -abs
	}

	avg := n.average
	avg3 := avg * 3

	avgMid := (avg * 4) / 3
	if !n.wide {
		avg3 = int64(int32(avg) * 3)
		avgMid = int64((int32(avg) * 4) / 3)
	}

	delta := int64(0)

	switch {
	case abs > avg3:
		delta = ((input >> 25) & 64) - deltaLarge
	case abs > avgMid:
		delta = ((input >> 26) & 32) - deltaMid
	case abs > 0:
		delta = ((input >> 27) & 16) - deltaSmall
	}

	n.delta.set(0, delta)
	n.average += (abs - n.average) / 16
	n.delta.set(-1, n.delta.get(-1)>>1)
	n.delta.set(-2, n.delta.get(-2)>>1)
	n.delta.set(-8, n.delta.get(-8)>>1)
}

func saturateInt16(v int64) int32 {
	if v >= -32768 && v <= 32767 {
		return int32(v)
	}

	if v < 0 {
		return -32768
	}

	return 32767
}

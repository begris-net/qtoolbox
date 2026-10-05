package ape

func encodeFrame(pcm []byte, channels, sampleBits int, level Compression) []byte {
	prep := prepare(pcm, channels, sampleBits)
	bits := newBitArray()
	bits.reset()
	bits.encodeU32(prep.crc)

	if prep.special != 0 {
		bits.encodeU32(uint32(prep.special))
	}

	bits.flushCoder()

	if channels > 2 {
		encodeMulti(bits, prep.ch, sampleBits, level)
	} else {
		predX := newPredictor(sampleBits, level)
		predY := newPredictor(sampleBits, level)

		var sumX, sumY uint32 = kSumStart, kSumStart
		encodeChannels(bits, prep, channels, predX, predY, &sumX, &sumY)
	}

	bits.finalize()

	for bits.bit%8 != 0 {
		bits.bit++
	}

	return bits.payload()
}

func encodeChannels(bits *bitArray, prep prepared, channels int, predX, predY *predictor, sumX, sumY *uint32) {
	if channels == 1 {
		encodeMono(bits, prep, predX, sumX)

		return
	}

	encodeX, encodeY := stereoFlags(prep.special)
	switch {
	case encodeX && encodeY:
		encodeStereo(bits, prep, predX, predY, sumX, sumY)
	case encodeX:
		encodeSeries(bits, prep.ch[0], predX, sumX)
	case encodeY:
		encodeSeries(bits, prep.ch[1], predY, sumY)
	}
}

func encodeMono(bits *bitArray, prep prepared, predX *predictor, sumX *uint32) {
	if prep.special&specialMonoSilence != 0 {
		return
	}

	encodeSeries(bits, prep.ch[0], predX, sumX)
}

func encodeMulti(bits *bitArray, channels [][]int32, sampleBits int, level Compression) {
	preds := make([]*predictor, len(channels))
	sums := make([]uint32, len(channels))

	for idx := range channels {
		preds[idx] = newPredictor(sampleBits, level)
		sums[idx] = kSumStart
	}

	for block := range channels[0] {
		for idx := range channels {
			bits.encodeValue(preds[idx].compress(channels[idx][block], 0), &sums[idx])
		}
	}
}

func stereoFlags(special int) (bool, bool) {
	encodeX := true
	encodeY := true

	if special&specialLeftSilence != 0 && special&specialRightSilence != 0 {
		encodeX = false
		encodeY = false
	}

	if special&specialPseudoStereo != 0 {
		encodeY = false
	}

	return encodeX, encodeY
}

func encodeSeries(bits *bitArray, samples []int32, pred *predictor, kSum *uint32) {
	for _, sample := range samples {
		bits.encodeValue(pred.compress(sample, 0), kSum)
	}
}

func encodeStereo(bits *bitArray, prep prepared, predX, predY *predictor, sumX, sumY *uint32) {
	var lastX int32

	for i := range prep.ch[0] {
		bits.encodeValue(predY.compress(prep.ch[1][i], lastX), sumY)
		bits.encodeValue(predX.compress(prep.ch[0][i], prep.ch[1][i]), sumX)
		lastX = prep.ch[0][i]
	}
}

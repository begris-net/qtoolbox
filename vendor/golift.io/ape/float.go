package ape

// transformFloat matches CFloatTransform::Process. The decoder reverses it
// after expanding the samples back to float bits.
func transformFloat(pcm []byte) []byte {
	for off := 0; off+wordSize <= len(pcm); off += wordSize {
		sample := uint32(pcm[off]) | uint32(pcm[off+1])<<8 | uint32(pcm[off+2])<<16 | uint32(pcm[off+3])<<24
		out := sample & 0xC3FFFFFF

		out |= ^(sample & 0x3C000000) ^ 0xC3FFFFFF
		if out&0x80000000 != 0 {
			out = ^out | 0x80000000
		}

		pcm[off] = byte(out)
		pcm[off+1] = byte(out >> 8)
		pcm[off+2] = byte(out >> 16)
		pcm[off+3] = byte(out >> 24)
	}

	return pcm
}

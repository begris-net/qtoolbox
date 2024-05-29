package util

import (
	"errors"
	"math"
)

func OrElse[T string](value T, defaultValue T) T {
	if len(value) > 0 {
		return value
	}
	return defaultValue
}

func SafeDeref[T any](p *T) T {
	if p == nil {
		var v T
		return v
	}
	return *p
}

func Chunks[T any](chunkSize int, slice []T) ([][]T, error) {
	if len(slice) == 0 || chunkSize > len(slice) {
		return nil, errors.New("nothing to be done here, check your chunk and content sizes")
	}

	numOfSlices := int(math.Ceil(float64(len(slice)) / float64(chunkSize)))

	var slices [][]T
	begin, end := 0, chunkSize

	for i := 0; i < numOfSlices; i++ {
		slices = append(slices, slice[begin:end])
		if end+chunkSize > len(slice) {
			begin, end = end, len(slice)
		} else {
			begin, end = end, end+chunkSize
		}
	}

	return slices, nil
}

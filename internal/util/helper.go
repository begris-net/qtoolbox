/*
 * Copyright (c) 2024 Bjoern Beier.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy of
 * this software and associated documentation files (the "Software"), to deal in
 * the Software without restriction, including without limitation the rights to
 * use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
 * the Software, and to permit persons to whom the Software is furnished to do so,
 * subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
 * FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
 * COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
 * IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
 * CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 */

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

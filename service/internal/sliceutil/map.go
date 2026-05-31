package sliceutil

import "github.com/TrebuchetDynamics/goncho/service/internal/sliceutil/transformops"

// Map returns a slice containing fn applied to each value while preserving nil
// input as nil. A nil fn returns the zero value for each input element.
func Map[T any, U any](values []T, fn func(T) U) []U {
	return transformops.Map(values, fn)
}

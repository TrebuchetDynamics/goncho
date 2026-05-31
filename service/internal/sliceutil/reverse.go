package sliceutil

import "github.com/TrebuchetDynamics/goncho/service/internal/sliceutil/copyops"

// ReverseClone returns a new slice with values from in in reverse order.
func ReverseClone[T any](in []T) []T {
	return copyops.ReverseClone(in)
}

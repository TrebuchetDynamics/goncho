package sliceutil

import "github.com/TrebuchetDynamics/goncho/service/internal/sliceutil/copyops"

// Clone returns a shallow copy of a slice while preserving nil input as nil.
func Clone[T any](in []T) []T {
	return copyops.Clone(in)
}

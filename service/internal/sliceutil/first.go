package sliceutil

import "github.com/TrebuchetDynamics/goncho/service/internal/sliceutil/searchops"

// First returns the first value from values, or the zero value when values is empty.
func First[T any](values []T) T {
	return searchops.First(values)
}

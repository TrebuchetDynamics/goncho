package sliceutil

import "github.com/TrebuchetDynamics/goncho/service/internal/sliceutil/transformops"

// IndexBy builds an index of the first position for each accepted key in values.
func IndexBy[T any, K comparable](values []T, key func(T) (K, bool)) map[K]int {
	return transformops.IndexBy(values, key)
}

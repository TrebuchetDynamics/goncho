package sliceutil

import "github.com/TrebuchetDynamics/goncho/service/internal/sliceutil/searchops"

// Contains reports whether values includes want.
func Contains[T comparable](values []T, want T) bool {
	return searchops.Contains(values, want)
}

// ContainsFunc reports whether any value satisfies match.
func ContainsFunc[T any](values []T, match func(T) bool) bool {
	return searchops.ContainsFunc(values, match)
}

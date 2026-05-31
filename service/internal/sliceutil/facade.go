package sliceutil

import (
	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil/copyops"
	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil/limitops"
	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil/searchops"
	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil/transformops"
)

// Clone returns a shallow copy of a slice while preserving nil input as nil.
func Clone[T any](in []T) []T {
	return copyops.Clone(in)
}

// ReverseClone returns a new slice with values from in in reverse order.
func ReverseClone[T any](in []T) []T {
	return copyops.ReverseClone(in)
}

// Contains reports whether values includes want.
func Contains[T comparable](values []T, want T) bool {
	return searchops.Contains(values, want)
}

// ContainsFunc reports whether any value satisfies match.
func ContainsFunc[T any](values []T, match func(T) bool) bool {
	return searchops.ContainsFunc(values, match)
}

// First returns the first value from values, or the zero value when values is empty.
func First[T any](values []T) T {
	return searchops.First(values)
}

// Map returns a slice containing fn applied to each value while preserving nil
// input as nil. A nil fn returns the zero value for each input element.
func Map[T any, U any](values []T, fn func(T) U) []U {
	return transformops.Map(values, fn)
}

// FilterMap returns mapped values for inputs accepted by fn while preserving nil
// input as nil. A nil fn rejects every value.
func FilterMap[T any, U any](values []T, fn func(T) (U, bool)) []U {
	return transformops.FilterMap(values, fn)
}

// IndexBy builds an index of the first position for each accepted key in values.
func IndexBy[T any, K comparable](values []T, key func(T) (K, bool)) map[K]int {
	return transformops.IndexBy(values, key)
}

// Limit returns values truncated to at most limit items. A non-positive limit
// means no truncation, matching call sites where zero is an unset limit.
func Limit[T any](values []T, limit int) []T {
	return limitops.Limit(values, limit)
}

// LimitClone returns a shallow copy truncated to at most limit items. A
// non-positive limit means no truncation.
func LimitClone[T any](values []T, limit int) []T {
	return limitops.LimitClone(values, limit)
}

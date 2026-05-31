package sliceutil

import "github.com/TrebuchetDynamics/goncho/service/internal/sliceutil/transformops"

// FilterMap returns mapped values for inputs accepted by fn while preserving nil
// input as nil. A nil fn rejects every value.
func FilterMap[T any, U any](values []T, fn func(T) (U, bool)) []U {
	return transformops.FilterMap(values, fn)
}

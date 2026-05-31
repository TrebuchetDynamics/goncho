package stringlist

import "github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"

// Clone returns a shallow copy of values.
func Clone(values []string) []string {
	return sliceutil.Clone(values)
}

// Predicate is the shared contract for matching a single string slice item.
type Predicate func(value string) bool

// Any reports whether match accepts at least one value. A nil predicate never
// matches.
func Any(values []string, match Predicate) bool {
	if match == nil {
		return false
	}
	for _, value := range values {
		if match(value) {
			return true
		}
	}
	return false
}

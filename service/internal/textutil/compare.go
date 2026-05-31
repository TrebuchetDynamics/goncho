package textutil

import "github.com/TrebuchetDynamics/goncho/service/internal/textutil/trimmed"

// EqualTrimmed reports whether two strings are equal after trimming ASCII/
// Unicode whitespace.
func EqualTrimmed(a, b string) bool {
	return trimmed.Equal(a, b)
}

// EqualFoldTrimmed reports whether two strings are equal after trimming ASCII/
// Unicode whitespace and applying Unicode case-folding.
func EqualFoldTrimmed(a, b string) bool {
	return trimmed.EqualFold(a, b)
}

// ContainsTrimmed reports whether values contains want after trimming ASCII/
// Unicode whitespace on both sides.
func ContainsTrimmed(values []string, want string) bool {
	for _, value := range values {
		if EqualTrimmed(value, want) {
			return true
		}
	}
	return false
}

// ContainsEqualFoldTrimmed reports whether values contains want after trimming
// ASCII/Unicode whitespace and applying Unicode case-folding.
func ContainsEqualFoldTrimmed(values []string, want string) bool {
	for _, value := range values {
		if EqualFoldTrimmed(value, want) {
			return true
		}
	}
	return false
}

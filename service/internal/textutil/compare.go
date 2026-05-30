package textutil

import "strings"

// EqualTrimmed reports whether two strings are equal after trimming ASCII/
// Unicode whitespace.
func EqualTrimmed(a, b string) bool {
	return strings.TrimSpace(a) == strings.TrimSpace(b)
}

// EqualFoldTrimmed reports whether two strings are equal after trimming ASCII/
// Unicode whitespace and applying Unicode case-folding.
func EqualFoldTrimmed(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
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

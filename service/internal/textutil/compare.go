package textutil

import "strings"

// EqualFoldTrimmed reports whether two strings are equal after trimming ASCII/
// Unicode whitespace and applying Unicode case-folding.
func EqualFoldTrimmed(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

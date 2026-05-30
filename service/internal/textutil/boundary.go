package textutil

import "strings"

// CutBeforeAnySubstringFold returns value before the first matching marker,
// using case-insensitive matching. Empty markers are ignored.
func CutBeforeAnySubstringFold(value string, markers ...string) (string, bool) {
	lower := strings.ToLower(value)
	best := -1
	for _, marker := range markers {
		if marker == "" {
			continue
		}
		idx := strings.Index(lower, strings.ToLower(marker))
		if idx < 0 {
			continue
		}
		if best < 0 || idx < best {
			best = idx
		}
	}
	if best < 0 {
		return value, false
	}
	return value[:best], true
}

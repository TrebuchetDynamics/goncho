package substrmatch

import "strings"

// Any reports whether value contains at least one marker using case-sensitive
// matching. Empty marker lists do not match.
func Any(value string, markers []string) bool {
	for _, marker := range markers {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return false
}

// Either reports whether either value contains the other using case-sensitive
// matching.
func Either(a, b string) bool {
	return strings.Contains(a, b) || strings.Contains(b, a)
}

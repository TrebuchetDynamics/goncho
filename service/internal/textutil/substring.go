package textutil

import "strings"

// ContainsAnySubstring reports whether value contains at least one marker.
func ContainsAnySubstring(value string, markers []string) bool {
	for _, marker := range markers {
		if strings.Contains(value, marker) {
			return true
		}
	}
	return false
}

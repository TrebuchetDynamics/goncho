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

// ContainsAnySubstringFold reports whether value contains at least one marker,
// comparing with the same simple case-fold policy used by Goncho text filters.
func ContainsAnySubstringFold(value string, markers []string) bool {
	return ContainsAnySubstring(strings.ToLower(value), lowerStrings(markers))
}

func lowerStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	for i, value := range values {
		out[i] = strings.ToLower(value)
	}
	return out
}

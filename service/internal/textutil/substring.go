package textutil

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/textutil/foldmatch"
)

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
	return foldmatch.AnySubstring(value, markers)
}

// ContainsEitherSubstring reports whether either value contains the other.
func ContainsEitherSubstring(a, b string) bool {
	return strings.Contains(a, b) || strings.Contains(b, a)
}

// ContainsEitherSubstringFold reports whether either value contains the other
// after applying the same simple case-fold policy used by Goncho text filters.
func ContainsEitherSubstringFold(a, b string) bool {
	return ContainsEitherSubstring(strings.ToLower(a), strings.ToLower(b))
}

// ContainsAllSubstringsFold reports whether value contains every non-blank
// marker after trimming markers and applying the same simple case-fold policy
// used by Goncho text filters. Blank markers are ignored.
func ContainsAllSubstringsFold(value string, markers []string) bool {
	return foldmatch.AllSubstrings(value, markers)
}

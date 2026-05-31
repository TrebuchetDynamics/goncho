package textutil

import (
	"github.com/TrebuchetDynamics/goncho/service/internal/textutil/foldmatch"
	"github.com/TrebuchetDynamics/goncho/service/internal/textutil/substrmatch"
)

// ContainsAnySubstring reports whether value contains at least one marker.
func ContainsAnySubstring(value string, markers []string) bool {
	return substrmatch.Any(value, markers)
}

// ContainsAnySubstringFold reports whether value contains at least one marker,
// comparing with the same simple case-fold policy used by Goncho text filters.
func ContainsAnySubstringFold(value string, markers []string) bool {
	return foldmatch.AnySubstring(value, markers)
}

// ContainsEitherSubstring reports whether either value contains the other.
func ContainsEitherSubstring(a, b string) bool {
	return substrmatch.Either(a, b)
}

// ContainsEitherSubstringFold reports whether either value contains the other
// after applying the same simple case-fold policy used by Goncho text filters.
func ContainsEitherSubstringFold(a, b string) bool {
	return foldmatch.EitherSubstring(a, b)
}

// ContainsAllSubstringsFold reports whether value contains every non-blank
// marker after trimming markers and applying the same simple case-fold policy
// used by Goncho text filters. Blank markers are ignored.
func ContainsAllSubstringsFold(value string, markers []string) bool {
	return foldmatch.AllSubstrings(value, markers)
}

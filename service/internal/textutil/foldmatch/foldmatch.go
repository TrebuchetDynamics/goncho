package foldmatch

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/textutil/foldcase"
)

// AnySubstring reports whether value contains at least one marker using simple
// lower-case matching. Markers are compared as provided; empty markers therefore
// match the same way strings.Contains does.
func AnySubstring(value string, markers []string) bool {
	value = foldcase.Lower(value)
	for _, marker := range markers {
		if foldcase.ContainsFolded(value, marker) {
			return true
		}
	}
	return false
}

// AllSubstrings reports whether value contains every non-blank marker using
// simple lower-case matching after trimming markers.
func AllSubstrings(value string, markers []string) bool {
	value = foldcase.Lower(value)
	for _, marker := range markers {
		marker = foldcase.Lower(strings.TrimSpace(marker))
		if marker == "" {
			continue
		}
		if !strings.Contains(value, marker) {
			return false
		}
	}
	return true
}

// HasAnyPrefix reports whether value starts with any non-empty prefix using
// simple lower-case matching.
func HasAnyPrefix(value string, prefixes ...string) bool {
	value = foldcase.Lower(value)
	for _, prefix := range prefixes {
		if prefix == "" {
			continue
		}
		if foldcase.HasPrefixFolded(value, prefix) {
			return true
		}
	}
	return false
}

// CutAnyPrefix removes the first matching prefix using simple lower-case
// matching. The returned tail preserves original casing and spacing from value.
func CutAnyPrefix(value string, prefixes []string) (tail string, ok bool) {
	lower := foldcase.Lower(value)
	for _, prefix := range prefixes {
		if foldcase.HasPrefixFolded(lower, prefix) {
			return value[len(prefix):], true
		}
	}
	return "", false
}

// CutAroundAnySubstringMatch splits value around the first matching marker
// using simple lower-case matching. Returned parts preserve original casing and
// spacing from value; marker is the policy marker that matched.
func CutAroundAnySubstringMatch(value string, markers []string) (before, marker, after string, ok bool) {
	lower := foldcase.Lower(value)
	for _, candidate := range markers {
		idx := foldcase.IndexFolded(lower, candidate)
		if idx < 0 {
			continue
		}
		return value[:idx], candidate, value[idx+len(candidate):], true
	}
	return "", "", "", false
}

// CutBeforeAnySubstring returns value before the earliest matching non-empty
// marker using simple lower-case matching.
func CutBeforeAnySubstring(value string, markers ...string) (string, bool) {
	lower := foldcase.Lower(value)
	best := -1
	for _, marker := range markers {
		if marker == "" {
			continue
		}
		idx := foldcase.IndexFolded(lower, marker)
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

// EitherSubstring reports whether either value contains the other using simple
// lower-case matching.
func EitherSubstring(a, b string) bool {
	return foldcase.EitherSubstring(a, b)
}

package textutil

import "strings"

// CutAnyPrefixFold removes the first matching prefix using the same simple
// case-fold policy as Goncho text classifiers. The returned tail preserves the
// original casing and spacing from value.
func CutAnyPrefixFold(value string, prefixes []string) (tail string, ok bool) {
	lower := strings.ToLower(value)
	for _, prefix := range prefixes {
		if strings.HasPrefix(lower, strings.ToLower(prefix)) {
			return value[len(prefix):], true
		}
	}
	return "", false
}

// CutAroundAnySubstringFold splits value around the first matching marker using
// simple case-folding. The returned parts preserve the original casing and
// spacing from value.
func CutAroundAnySubstringFold(value string, markers []string) (before, after string, ok bool) {
	before, _, after, ok = CutAroundAnySubstringFoldMatch(value, markers)
	return before, after, ok
}

// CutAroundAnySubstringFoldMatch is like CutAroundAnySubstringFold and also
// returns the matching policy marker.
func CutAroundAnySubstringFoldMatch(value string, markers []string) (before, marker, after string, ok bool) {
	lower := strings.ToLower(value)
	for _, candidate := range markers {
		idx := strings.Index(lower, strings.ToLower(candidate))
		if idx < 0 {
			continue
		}
		return value[:idx], candidate, value[idx+len(candidate):], true
	}
	return "", "", "", false
}

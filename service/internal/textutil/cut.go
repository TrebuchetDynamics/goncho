package textutil

import "github.com/TrebuchetDynamics/goncho/service/internal/textutil/foldmatch"

// CutAnyPrefixFold removes the first matching prefix using the same simple
// case-fold policy as Goncho text classifiers. The returned tail preserves the
// original casing and spacing from value.
func CutAnyPrefixFold(value string, prefixes []string) (tail string, ok bool) {
	return foldmatch.CutAnyPrefix(value, prefixes)
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
	return foldmatch.CutAroundAnySubstringMatch(value, markers)
}

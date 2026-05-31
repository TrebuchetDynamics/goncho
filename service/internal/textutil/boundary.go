package textutil

import "github.com/TrebuchetDynamics/goncho/service/internal/textutil/foldmatch"

// CutBeforeAnySubstringFold returns value before the first matching marker,
// using case-insensitive matching. Empty markers are ignored.
func CutBeforeAnySubstringFold(value string, markers ...string) (string, bool) {
	return foldmatch.CutBeforeAnySubstring(value, markers...)
}

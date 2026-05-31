package textutil

import "github.com/TrebuchetDynamics/goncho/service/internal/textutil/foldmatch"

// HasAnyPrefixFold reports whether value starts with any prefix,
// case-insensitively. Empty prefixes are ignored.
func HasAnyPrefixFold(value string, prefixes ...string) bool {
	return foldmatch.HasAnyPrefix(value, prefixes...)
}

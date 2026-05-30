package textutil

import "strings"

// HasAnyPrefixFold reports whether value starts with any prefix,
// case-insensitively. Empty prefixes are ignored.
func HasAnyPrefixFold(value string, prefixes ...string) bool {
	lower := strings.ToLower(value)
	for _, prefix := range prefixes {
		if prefix == "" {
			continue
		}
		if strings.HasPrefix(lower, strings.ToLower(prefix)) {
			return true
		}
	}
	return false
}

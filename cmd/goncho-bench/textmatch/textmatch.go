package textmatch

import "strings"

// ContainsAny reports whether value contains at least one candidate substring.
func ContainsAny(value string, candidates []string) bool {
	for _, candidate := range candidates {
		if strings.Contains(value, candidate) {
			return true
		}
	}
	return false
}

package textutil

import (
	"sort"
	"strings"
)

// NormalizeUnique returns non-empty normalized strings, preserving first-seen
// order unless sortOutput is true.
func NormalizeUnique(values []string, normalize func(string) string, sortOutput bool) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if normalize != nil {
			value = normalize(value)
		}
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	if len(out) == 0 {
		return nil
	}
	if sortOutput {
		sort.Strings(out)
	}
	return out
}

// UniqueTrimmed returns distinct non-empty strings after whitespace trimming.
func UniqueTrimmed(values []string, sortOutput bool) []string {
	return NormalizeUnique(values, strings.TrimSpace, sortOutput)
}

// UniqueLowerTrimmed returns distinct non-empty strings after trimming and
// lower-casing.
func UniqueLowerTrimmed(values []string, sortOutput bool) []string {
	return NormalizeUnique(values, func(value string) string {
		return strings.ToLower(strings.TrimSpace(value))
	}, sortOutput)
}

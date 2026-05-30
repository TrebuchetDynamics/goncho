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
	return NormalizeUnique(values, lowerTrimmed, sortOutput)
}

// Set returns normalized non-empty strings as a set. It preserves nil for empty
// input or when every normalized value is empty.
func Set(values []string, normalize func(string) string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		if normalize != nil {
			value = normalize(value)
		}
		if value == "" {
			continue
		}
		out[value] = struct{}{}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// TrimmedSet returns distinct non-empty strings after whitespace trimming.
func TrimmedSet(values []string) map[string]struct{} {
	return Set(values, strings.TrimSpace)
}

// LowerTrimmedSet returns distinct non-empty strings after trimming and
// lower-casing.
func LowerTrimmedSet(values []string) map[string]struct{} {
	return Set(values, lowerTrimmed)
}

// SortedSetValues returns the sorted non-empty keys in values after optional
// normalization.
func SortedSetValues(values map[string]struct{}, normalize func(string) string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for value := range values {
		if normalize != nil {
			value = normalize(value)
		}
		if value != "" {
			out = append(out, value)
		}
	}
	if len(out) == 0 {
		return nil
	}
	if len(out) > 1 {
		sort.Strings(out)
	}
	return out
}

func lowerTrimmed(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

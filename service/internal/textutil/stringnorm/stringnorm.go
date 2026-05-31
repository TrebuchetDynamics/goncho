package stringnorm

import "sort"

// Normalizer is the shared contract for callers that canonicalize string values
// before set/unique operations. A nil Normalizer preserves values as-is.
type Normalizer func(string) string

// Apply runs normalize when present and otherwise returns value unchanged.
func Apply(normalize Normalizer, value string) string {
	if normalize == nil {
		return value
	}
	return normalize(value)
}

// Unique returns non-empty normalized strings, preserving first-seen order
// unless sortOutput is true.
func Unique(values []string, normalize Normalizer, sortOutput bool) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = Apply(normalize, value)
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

// Set returns normalized non-empty strings as a set. It preserves nil for empty
// input or when every normalized value is empty.
func Set(values []string, normalize Normalizer) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = Apply(normalize, value)
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

// SortedSetValues returns the sorted non-empty keys in values after optional
// normalization.
func SortedSetValues(values map[string]struct{}, normalize Normalizer) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for value := range values {
		value = Apply(normalize, value)
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

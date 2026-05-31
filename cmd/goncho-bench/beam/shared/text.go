package shared

import "strings"

// HasNonEmptyTrimmed reports whether value has non-empty content after trimming.
func HasNonEmptyTrimmed(value string) bool {
	return strings.TrimSpace(value) != ""
}

// FirstNonEmptyTrimmed returns the first argument with non-empty trimmed content.
func FirstNonEmptyTrimmed(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

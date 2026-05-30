package stringutil

import "strings"

// FirstNonEmpty returns the first non-blank (after trimming whitespace) value
// from the arguments, or "" if all are blank.
func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

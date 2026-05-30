package pathutil

import (
	"path/filepath"
	"strings"
)

// CleanRelative returns a cleaned, non-empty relative path and rejects paths
// that would escape their root. It intentionally mirrors the service's
// conservative historical check where any cleaned ".." prefix is unsafe.
func CleanRelative(value string) (string, bool) {
	clean := filepath.Clean(strings.TrimSpace(value))
	if clean == "." || IsUnsafeRelative(clean) {
		return "", false
	}
	return clean, true
}

// IsUnsafeRelative reports whether a path is absolute or escapes upward from a
// scoped root after cleaning.
func IsUnsafeRelative(value string) bool {
	clean := filepath.Clean(strings.TrimSpace(value))
	return strings.HasPrefix(clean, "..") || filepath.IsAbs(clean)
}

// CleanSlashPath returns a cleaned slash-separated path for stable API output.
func CleanSlashPath(value string) string {
	return filepath.ToSlash(filepath.Clean(strings.TrimSpace(value)))
}

// NormalizeSlashPattern trims whitespace, converts separators to slashes, and
// removes a leading ./ prefix from user-facing glob/path patterns.
func NormalizeSlashPattern(value string) string {
	value = filepath.ToSlash(strings.TrimSpace(value))
	return strings.TrimPrefix(value, "./")
}

// SlashBase returns the final path element from a slash-separated relative path.
func SlashBase(value string) string {
	idx := strings.LastIndex(value, "/")
	if idx < 0 {
		return value
	}
	return value[idx+1:]
}

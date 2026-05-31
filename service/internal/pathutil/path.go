package pathutil

import (
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

// AbsNonBlank trims value and resolves it to an absolute path.
func AbsNonBlank(value string) (string, bool, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false, nil
	}
	abs, err := filepath.Abs(trimmed)
	if err != nil {
		return "", false, err
	}
	return abs, true, nil
}

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

// NormalizeSlashPatterns trims, slash-normalizes, de-duplicates, and drops
// empty user-facing glob/path patterns while preserving first-seen order.
func NormalizeSlashPatterns(values []string) []string {
	return textutil.NormalizeUnique(values, NormalizeSlashPattern, false)
}

// MatchesAnySlashGlob reports whether rel matches at least one slash-separated
// glob pattern using Goncho's filesystem watcher matching semantics.
func MatchesAnySlashGlob(rel string, patterns []string) bool {
	for _, pattern := range patterns {
		if MatchSlashGlob(rel, pattern) {
			return true
		}
	}
	return false
}

// MatchSlashGlob matches a slash-separated relative path against a user-facing
// glob/path pattern. It supports direct path/base matches, *, **, prefix/**,
// and **/suffix forms in addition to filepath.Match checks.
func MatchSlashGlob(rel, pattern string) bool {
	rel = NormalizeSlashPattern(rel)
	pattern = NormalizeSlashPattern(pattern)
	base := SlashBase(rel)
	if pattern == rel || pattern == base || pattern == "**" || pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return rel == prefix || strings.HasPrefix(rel, prefix+"/")
	}
	if strings.HasPrefix(pattern, "**/") {
		tail := strings.TrimPrefix(pattern, "**/")
		if ok, _ := filepath.Match(tail, base); ok {
			return true
		}
		return strings.HasSuffix(rel, strings.TrimPrefix(tail, "*"))
	}
	if ok, _ := filepath.Match(pattern, rel); ok {
		return true
	}
	if ok, _ := filepath.Match(pattern, base); ok {
		return true
	}
	return false
}

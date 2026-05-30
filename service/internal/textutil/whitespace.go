package textutil

import (
	"strings"
)

// CollapseWhitespace trims leading/trailing whitespace and converts any run of
// Unicode whitespace to a single ASCII space.
func CollapseWhitespace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

// FirstWords returns the first n whitespace-delimited words from content. When
// content has n or fewer words, it preserves the caller-visible trimmed text
// instead of rebuilding spacing between words.
func FirstWords(content string, n int) string {
	words := strings.Fields(content)
	if n <= 0 {
		return ""
	}
	if len(words) <= n {
		return strings.TrimSpace(content)
	}
	return strings.Join(words[:n], " ")
}

// CompactWhitespace collapses whitespace and limits the result to limit bytes,
// trimming a partial trailing word/space boundary the same way existing preview
// callers historically did.
func CompactWhitespace(value string, limit int, empty string) string {
	value = CollapseWhitespace(value)
	if value == "" {
		return empty
	}
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return strings.TrimSpace(value[:limit])
}

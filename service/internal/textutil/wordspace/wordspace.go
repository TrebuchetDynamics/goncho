package wordspace

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/textutil/trimmed"
)

// Words applies Goncho's whitespace-tokenization policy.
func Words(value string) []string {
	return strings.Fields(value)
}

// Collapse trims leading/trailing whitespace and converts any run of Unicode
// whitespace to a single ASCII space.
func Collapse(value string) string {
	return strings.Join(Words(value), " ")
}

// FirstWords returns the first n whitespace-delimited words from content. When
// content has n or fewer words, it preserves the caller-visible trimmed text
// instead of rebuilding spacing between words.
func FirstWords(content string, n int) string {
	words := Words(content)
	if n <= 0 {
		return ""
	}
	if len(words) <= n {
		return trimmed.Space(content)
	}
	return strings.Join(words[:n], " ")
}

// WordCount returns the number of whitespace-delimited words in content.
func WordCount(content string) int {
	return len(Words(content))
}

// ApproxTokens returns Goncho's stable, low-cost token estimate for budgeting.
// Blank content is treated as one token so callers never undercount an empty
// but present field.
func ApproxTokens(content string) int {
	content = trimmed.Space(content)
	if content == "" {
		return 1
	}
	if n := WordCount(content); n > 0 {
		return n
	}
	return 1
}

// FitsTokenBudget reports whether an item with cost can be added after used.
// When allowFirstOverBudget is true, the first item is admitted even when it
// exceeds the budget so callers can return at least one relevant result.
func FitsTokenBudget(used, cost, budget int, allowFirstOverBudget bool) bool {
	if budget <= 0 {
		return false
	}
	if allowFirstOverBudget && used == 0 {
		return true
	}
	return used+cost <= budget
}

// Compact collapses whitespace and limits the result to limit bytes, trimming a
// partial trailing word/space boundary the same way existing preview callers
// historically did.
func Compact(value string, limit int, empty string) string {
	value = Collapse(value)
	if value == "" {
		return empty
	}
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return trimmed.Space(value[:limit])
}

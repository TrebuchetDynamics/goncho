package textutil

import "github.com/TrebuchetDynamics/goncho/service/internal/textutil/wordspace"

// CollapseWhitespace trims leading/trailing whitespace and converts any run of
// Unicode whitespace to a single ASCII space.
func CollapseWhitespace(value string) string {
	return wordspace.Collapse(value)
}

// FirstWords returns the first n whitespace-delimited words from content. When
// content has n or fewer words, it preserves the caller-visible trimmed text
// instead of rebuilding spacing between words.
func FirstWords(content string, n int) string {
	return wordspace.FirstWords(content, n)
}

// WordCount returns the number of whitespace-delimited words in content.
func WordCount(content string) int {
	return wordspace.WordCount(content)
}

// ApproxTokens returns Goncho's stable, low-cost token estimate for budgeting.
// Blank content is treated as one token so callers never undercount an empty
// but present field.
func ApproxTokens(content string) int {
	return wordspace.ApproxTokens(content)
}

// FitsTokenBudget reports whether an item with cost can be added after used.
// When allowFirstOverBudget is true, the first item is admitted even when it
// exceeds the budget so callers can return at least one relevant result.
func FitsTokenBudget(used, cost, budget int, allowFirstOverBudget bool) bool {
	return wordspace.FitsTokenBudget(used, cost, budget, allowFirstOverBudget)
}

// CompactWhitespace collapses whitespace and limits the result to limit bytes,
// trimming a partial trailing word/space boundary the same way existing preview
// callers historically did.
func CompactWhitespace(value string, limit int, empty string) string {
	return wordspace.Compact(value, limit, empty)
}

package textutil

import "strings"

const boundaryQuoteChars = "\"'`“”‘’"

// TrimSpaceAndQuotes trims surrounding whitespace, then removes quote-like
// boundary characters used by fact extraction and prompt classifiers.
func TrimSpaceAndQuotes(value string) string {
	return strings.Trim(strings.TrimSpace(value), boundaryQuoteChars)
}

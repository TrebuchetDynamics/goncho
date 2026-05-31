package textutil

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/textutil/trimmed"
)

const boundaryQuoteChars = "\"'`“”‘’"

// TrimSpaceAndQuotes trims surrounding whitespace, then removes quote-like
// boundary characters used by fact extraction and prompt classifiers.
func TrimSpaceAndQuotes(value string) string {
	return strings.Trim(trimmed.Space(value), boundaryQuoteChars)
}

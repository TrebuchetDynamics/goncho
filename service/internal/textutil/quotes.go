package textutil

import "github.com/TrebuchetDynamics/goncho/service/internal/textutil/trimmed"

// TrimSpaceAndQuotes trims surrounding whitespace, then removes quote-like
// boundary characters used by fact extraction and prompt classifiers.
func TrimSpaceAndQuotes(value string) string {
	return trimmed.SpaceAndQuotes(value)
}

package trimmed

import "strings"

const boundaryQuoteChars = "\"'`“”‘’"

// Space trims surrounding Unicode whitespace using Goncho's shared boundary
// normalization policy.
func Space(value string) string {
	return strings.TrimSpace(value)
}

// Blank reports whether value is empty after applying Space.
func Blank(value string) bool {
	return Space(value) == ""
}

// NonBlank reports whether value has non-whitespace content.
func NonBlank(value string) bool {
	return !Blank(value)
}

// FirstNonBlank returns the first value with non-whitespace content after
// applying Space. It returns an empty string when every value is blank.
func FirstNonBlank(values ...string) string {
	for _, value := range values {
		if value := Space(value); value != "" {
			return value
		}
	}
	return ""
}

// Lower applies Space and simple lower-casing.
func Lower(value string) string {
	return strings.ToLower(Space(value))
}

// Upper applies Space and simple upper-casing.
func Upper(value string) string {
	return strings.ToUpper(Space(value))
}

// SpaceAndQuotes trims surrounding whitespace, then removes quote-like boundary
// characters used by fact extraction and prompt classifiers.
func SpaceAndQuotes(value string) string {
	return strings.Trim(Space(value), boundaryQuoteChars)
}

// SentenceBoundary removes sentence punctuation, then trims surrounding
// whitespace. It intentionally keeps punctuation inside the value unchanged.
func SentenceBoundary(value string) string {
	return Space(strings.Trim(value, ".!?"))
}

// QuestionPunctuation removes leading/trailing question punctuation before
// trimming whitespace.
func QuestionPunctuation(value string) string {
	return Space(strings.Trim(value, "?!."))
}

// QuestionPhraseBoundary removes question punctuation, dots, and spaces as
// boundary characters.
func QuestionPhraseBoundary(value string) string {
	return Space(strings.Trim(value, "?! ."))
}

// Equal reports whether two values match after applying Space.
func Equal(a, b string) bool {
	return Space(a) == Space(b)
}

// EqualFold reports whether two values match after applying Space and Unicode
// case-folding.
func EqualFold(a, b string) bool {
	return strings.EqualFold(Space(a), Space(b))
}

// Contains reports whether values contains want after applying Space to both
// sides of each comparison.
func Contains(values []string, want string) bool {
	for _, value := range values {
		if Equal(value, want) {
			return true
		}
	}
	return false
}

// ContainsEqualFold reports whether values contains want after applying Space
// and Unicode case-folding to both sides of each comparison.
func ContainsEqualFold(values []string, want string) bool {
	for _, value := range values {
		if EqualFold(value, want) {
			return true
		}
	}
	return false
}

// OptionalMatch reports whether value satisfies an optional exact-match filter
// after trimming the filter. An empty filter matches every value.
func OptionalMatch(value, filter string) bool {
	filter = Space(filter)
	return filter == "" || value == filter
}

// OptionalMatchOrEmpty is OptionalMatch plus legacy-empty value admission.
func OptionalMatchOrEmpty(value, filter string) bool {
	filter = Space(filter)
	return filter == "" || value == "" || value == filter
}

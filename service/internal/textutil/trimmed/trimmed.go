package trimmed

import "strings"

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

// Lower applies Space and simple lower-casing.
func Lower(value string) string {
	return strings.ToLower(Space(value))
}

// Upper applies Space and simple upper-casing.
func Upper(value string) string {
	return strings.ToUpper(Space(value))
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

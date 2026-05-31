package textutil

import "strings"

// MatchesOptionalTrimmed reports whether value satisfies an optional exact-match
// filter after trimming the filter. An empty filter matches every value.
func MatchesOptionalTrimmed(value, filter string) bool {
	filter = strings.TrimSpace(filter)
	return filter == "" || value == filter
}

// MatchesOptionalTrimmedOrEmpty reports whether value satisfies an optional
// exact-match filter after trimming the filter, treating an empty value as
// legacy unscoped data that should not be excluded by the filter.
func MatchesOptionalTrimmedOrEmpty(value, filter string) bool {
	filter = strings.TrimSpace(filter)
	return filter == "" || value == "" || value == filter
}

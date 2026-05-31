package textutil

import "github.com/TrebuchetDynamics/goncho/service/internal/textutil/trimmed"

// MatchesOptionalTrimmed reports whether value satisfies an optional exact-match
// filter after trimming the filter. An empty filter matches every value.
func MatchesOptionalTrimmed(value, filter string) bool {
	return trimmed.OptionalMatch(value, filter)
}

// MatchesOptionalTrimmedOrEmpty reports whether value satisfies an optional
// exact-match filter after trimming the filter, treating an empty value as
// legacy unscoped data that should not be excluded by the filter.
func MatchesOptionalTrimmedOrEmpty(value, filter string) bool {
	return trimmed.OptionalMatchOrEmpty(value, filter)
}

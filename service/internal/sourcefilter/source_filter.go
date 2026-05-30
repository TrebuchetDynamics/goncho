package sourcefilter

import "github.com/TrebuchetDynamics/goncho/service/internal/textutil"

// Allows reports whether a normalized source allow-list permits sourceType.
// Empty allow-lists and wildcard entries permit all sources. When
// emptySourceAllowed is true, an empty sourceType is treated as a legacy match
// for callers that historically accepted untyped vector hits.
func Allows(sources []string, sourceType string, emptySourceAllowed bool) bool {
	if len(sources) == 0 || hasWildcard(sources) {
		return true
	}
	if textutil.EqualTrimmed(sourceType, "") {
		return emptySourceAllowed
	}
	return textutil.ContainsEqualFoldTrimmed(sources, sourceType)
}

func hasWildcard(values []string) bool {
	return textutil.ContainsTrimmed(values, "") || textutil.ContainsTrimmed(values, "*")
}

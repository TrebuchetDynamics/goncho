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

// AllowsKindOrOrigin reports whether a source allow-list permits either a
// storage/source kind (for example "turn") or an adapter origin source (for
// example "discord"). This keeps source-kind filters from being silently
// reinterpreted as adapter prefixes.
func AllowsKindOrOrigin(sources []string, kind, origin string, emptySourceAllowed bool) bool {
	if len(sources) == 0 || hasWildcard(sources) {
		return true
	}
	if Allows(sources, kind, emptySourceAllowed) {
		return true
	}
	return Allows(sources, origin, emptySourceAllowed)
}

func hasWildcard(values []string) bool {
	return textutil.ContainsTrimmed(values, "") || textutil.ContainsTrimmed(values, "*")
}

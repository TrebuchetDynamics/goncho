package sourcefilter

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

// Decision is the replayable source-filter outcome behind Allows.
// It makes wildcard and legacy empty-source matches explicit for callers/tests
// that need to distinguish an intentional all-source browse from a filtered hit.
type Decision struct {
	Allowed            bool
	Wildcard           bool
	EmptySourceMatched bool
	MatchedSource      string
}

// Decide reports the explicit match reason for a normalized source allow-list.
// Empty allow-lists and wildcard entries permit all sources. When
// emptySourceAllowed is true, an empty sourceType is treated as a legacy match
// for callers that historically accepted untyped vector hits.
func Decide(sources []string, sourceType string, emptySourceAllowed bool) Decision {
	if len(sources) == 0 || hasWildcard(sources) {
		return Decision{Allowed: true, Wildcard: true, MatchedSource: "*"}
	}
	if textutil.EqualTrimmed(sourceType, "") {
		return Decision{Allowed: emptySourceAllowed, EmptySourceMatched: emptySourceAllowed}
	}
	if !textutil.ContainsEqualFoldTrimmed(sources, sourceType) {
		return Decision{}
	}
	return Decision{Allowed: true, MatchedSource: strings.TrimSpace(sourceType)}
}

// Allows reports whether a normalized source allow-list permits sourceType.
func Allows(sources []string, sourceType string, emptySourceAllowed bool) bool {
	return Decide(sources, sourceType, emptySourceAllowed).Allowed
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

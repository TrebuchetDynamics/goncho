package sourcefilter

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

// Allows reports whether a normalized source allow-list permits sourceType.
// Empty allow-lists and wildcard entries permit all sources. When
// emptySourceAllowed is true, an empty sourceType is treated as a legacy match
// for callers that historically accepted untyped vector hits.
func Allows(sources []string, sourceType string, emptySourceAllowed bool) bool {
	if len(sources) == 0 || hasWildcard(sources) {
		return true
	}
	if strings.TrimSpace(sourceType) == "" {
		return emptySourceAllowed
	}
	for _, source := range sources {
		if textutil.EqualFoldTrimmed(source, sourceType) {
			return true
		}
	}
	return false
}

func hasWildcard(values []string) bool {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || trimmed == "*" {
			return true
		}
	}
	return false
}

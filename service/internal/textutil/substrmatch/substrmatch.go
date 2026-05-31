package substrmatch

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/textutil/stringlist"
)

// Matcher is the shared contract for testing whether value matches a marker.
type Matcher func(value, marker string) bool

// AnyMatch reports whether value matches at least one marker using match.
func AnyMatch(value string, markers []string, match Matcher) bool {
	return stringlist.Any(markers, func(marker string) bool {
		return match != nil && match(value, marker)
	})
}

// Any reports whether value contains at least one marker using case-sensitive
// matching. Empty marker lists do not match.
func Any(value string, markers []string) bool {
	return AnyMatch(value, markers, strings.Contains)
}

// Either reports whether either value contains the other using case-sensitive
// matching.
func Either(a, b string) bool {
	return strings.Contains(a, b) || strings.Contains(b, a)
}

package ranking

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/metrics"
)

// FirstRelevantRank returns the 1-based position of the first retrieved relevant ID, or 0 when none match.
func FirstRelevantRank(retrievedIDs, relevantIDs []string) int {
	relevant := StringSet(relevantIDs)
	for i, id := range retrievedIDs {
		if _, ok := relevant[id]; ok {
			return i + 1
		}
	}
	return 0
}

// RecallAtK returns unique relevant-ID recall among the first k retrieved IDs.
func RecallAtK(retrievedIDs, relevantIDs []string, k int) float64 {
	if len(relevantIDs) == 0 || k <= 0 {
		return 0
	}
	relevant := StringSet(relevantIDs)
	limit := k
	if len(retrievedIDs) < limit {
		limit = len(retrievedIDs)
	}
	found := map[string]struct{}{}
	for _, id := range retrievedIDs[:limit] {
		if _, ok := relevant[id]; ok {
			found[id] = struct{}{}
		}
	}
	return metrics.Round(float64(len(found)) / float64(len(relevant)))
}

// StringSet builds a set from non-empty trimmed values.
func StringSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out[value] = struct{}{}
		}
	}
	return out
}

// TopN returns a copy of at most n leading values.
func TopN(values []string, n int) []string {
	if n > len(values) {
		n = len(values)
	}
	return append([]string(nil), values[:n]...)
}

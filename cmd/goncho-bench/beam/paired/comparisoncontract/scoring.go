package comparisoncontract

import "github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/paired/comparisoncontract/score"

// ScoreSummary preserves the comparisoncontract scoring API while sharing the
// score.Summary contract with lower-level comparison helpers.
type ScoreSummary = score.Summary

// Winner preserves the comparisoncontract winner API while delegating to the
// focused score contract.
func Winner(baseScore, candidateScore float64) string {
	return score.Winner(baseScore, candidateScore)
}

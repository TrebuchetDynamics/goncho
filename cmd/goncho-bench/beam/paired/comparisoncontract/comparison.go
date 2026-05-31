package comparisoncontract

import "github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/paired/comparisoncontract/summary"

// Report preserves the root comparisoncontract report API while sharing the
// focused summary.Report contract with lower-level comparison helpers.
type Report = summary.Report

// Stats preserves the root comparisoncontract ability-stats API while sharing
// the focused summary.Stats contract.
type Stats = summary.Stats

// Row preserves the root comparisoncontract row API while sharing the focused
// summary.Row contract.
type Row = summary.Row

// SummarizeRows preserves the root comparisoncontract summary API while
// delegating to the focused summary contract.
func SummarizeRows(rows []Row, bootstrapSamples int, bootstrapSeed int64, effectSizeFloor float64) Report {
	return summary.SummarizeRows(rows, bootstrapSamples, bootstrapSeed, effectSizeFloor)
}

// StatsForRows preserves the root comparisoncontract stats API while delegating
// to the focused summary contract.
func StatsForRows(rows []Row, effectSizeFloor float64) Stats {
	return summary.StatsForRows(rows, effectSizeFloor)
}

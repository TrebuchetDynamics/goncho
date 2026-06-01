package serviceartifact

import (
	"time"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/serviceartifact/caseprojection"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/serviceartifact/results"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/serviceartifact/summary"
	goncho "github.com/TrebuchetDynamics/goncho/service"
)

// BuildPairedOutcomes projects recall cases into the shared paired-outcome contract.
func BuildPairedOutcomes(report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time, score CaseScoreFunc) []PairedOutcome {
	return caseprojection.BuildPairedOutcomes(report, configID, runStartedAt, score)
}

// BuildFailureAuditRows projects failed recall cases into the stable failure-audit contract.
func BuildFailureAuditRows(report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time) []FailureAuditRow {
	return caseprojection.BuildFailureAuditRows(report, configID, runStartedAt)
}

// BuildResults projects a recall report into the BEAM service results artifact contract.
func BuildResults(report goncho.RecallBenchmarkReport, opts ResultsOptions) ResultsFile {
	return results.Build(report, opts)
}

// BuildSummary projects a recall report into the BEAM service summary artifact contract.
func BuildSummary(report goncho.RecallBenchmarkReport, opts SummaryOptions) SummaryFile {
	return summary.Build(report, opts)
}

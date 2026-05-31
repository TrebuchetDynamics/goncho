package paired

import (
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/paired/comparison"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/paired/comparisoncontract"
)

type beamPairedComparisonReport = comparisoncontract.Report

type beamPairedComparisonStats = comparisoncontract.Stats

type beamPairedComparisonRow = comparisoncontract.Row

func RunPairedComparison(cfg Config) error {
	return comparison.Run(comparisonConfig(cfg))
}

func buildBeamPairedComparison(cfg Config) (beamPairedComparisonReport, error) {
	return comparison.Build(comparisonConfig(cfg))
}

func loadBeamPairedOutcomes(path string) ([]servicePairedOutcome, error) {
	return comparison.LoadOutcomes(path)
}

func summarizeBeamPairedComparison(rows []beamPairedComparisonRow, bootstrapSamples int, effectSizeFloor float64) beamPairedComparisonReport {
	return comparison.Summarize(rows, bootstrapSamples, effectSizeFloor)
}

func writeBeamPairedComparisonJSON(jsonOut, markdownOut string, report beamPairedComparisonReport) error {
	return comparison.WriteJSON(jsonOut, markdownOut, report)
}

func writeBeamPairedComparisonMarkdown(path, jsonPath string, report beamPairedComparisonReport) error {
	return comparison.WriteMarkdown(path, jsonPath, report)
}

func sortedBeamPairedAbilities(byAbility map[string]beamPairedComparisonStats) []string {
	return comparison.SortedAbilities(byAbility)
}

func comparisonConfig(cfg Config) comparison.Config {
	return comparison.Config{
		ComparePath:             cfg.ComparePath,
		BaselineConfigID:        cfg.BaselineConfigID,
		CandidateConfigID:       cfg.CandidateConfigID,
		CompareJSONOut:          cfg.CompareJSONOut,
		CompareMarkdownOut:      cfg.CompareMarkdownOut,
		CompareBootstrapSamples: cfg.CompareBootstrapSamples,
		CompareEffectSizeFloor:  cfg.CompareEffectSizeFloor,
	}
}

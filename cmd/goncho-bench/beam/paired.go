package beam

import "github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/paired"

type PairedConfig struct {
	ComparePath             string
	BaselineConfigID        string
	CandidateConfigID       string
	CompareJSONOut          string
	CompareMarkdownOut      string
	CompareBootstrapSamples int
	CompareEffectSizeFloor  float64
	ResultsIn               string
	ResultsOut              string
	ResultsConfigID         string
}

func RunPairedComparison(cfg PairedConfig) error {
	return paired.RunPairedComparison(pairedConfig(cfg))
}

func AppendPairedOutcomesFromResults(cfg PairedConfig) error {
	return paired.AppendPairedOutcomesFromResults(pairedConfig(cfg))
}

func pairedConfig(cfg PairedConfig) paired.Config {
	return paired.Config{
		ComparePath:             cfg.ComparePath,
		BaselineConfigID:        cfg.BaselineConfigID,
		CandidateConfigID:       cfg.CandidateConfigID,
		CompareJSONOut:          cfg.CompareJSONOut,
		CompareMarkdownOut:      cfg.CompareMarkdownOut,
		CompareBootstrapSamples: cfg.CompareBootstrapSamples,
		CompareEffectSizeFloor:  cfg.CompareEffectSizeFloor,
		ResultsIn:               cfg.ResultsIn,
		ResultsOut:              cfg.ResultsOut,
		ResultsConfigID:         cfg.ResultsConfigID,
	}
}

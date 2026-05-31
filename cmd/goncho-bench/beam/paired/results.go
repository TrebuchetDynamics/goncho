package paired

import "github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/paired/results"

func AppendPairedOutcomesFromResults(cfg Config) error {
	return results.AppendOutcomesFromResults(resultsConfig(cfg))
}

func resultsConfig(cfg Config) results.Config {
	return results.Config{
		ResultsIn:       cfg.ResultsIn,
		ResultsOut:      cfg.ResultsOut,
		ResultsConfigID: cfg.ResultsConfigID,
	}
}

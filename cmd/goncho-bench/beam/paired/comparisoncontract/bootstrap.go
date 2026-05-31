package comparisoncontract

import "github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/paired/comparisoncontract/bootstrap"

// BootstrapMeanCI preserves the comparisoncontract bootstrap API while sharing
// the focused bootstrap.MeanCI helper with lower-level comparison code.
func BootstrapMeanCI(values []float64, samples int, seed int64) CI {
	return bootstrap.MeanCI(values, samples, seed)
}

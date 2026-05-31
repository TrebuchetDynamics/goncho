package score

import "github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/metrics"

func RoundMetric(v float64) float64 {
	return metrics.Round(v)
}

func RoundSignedMetric(v float64) float64 {
	return metrics.RoundSigned(v)
}

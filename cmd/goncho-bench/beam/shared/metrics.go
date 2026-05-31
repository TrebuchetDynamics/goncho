package shared

import "github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared/score"

func RoundMetric(v float64) float64 {
	return score.RoundMetric(v)
}

func RoundSignedMetric(v float64) float64 {
	return score.RoundSignedMetric(v)
}

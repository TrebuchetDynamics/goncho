package shared

import (
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/metrics"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/ranking"
)

func RoundMetric(v float64) float64 {
	return metrics.Round(v)
}

func TopN(values []string, n int) []string {
	return ranking.TopN(values, n)
}

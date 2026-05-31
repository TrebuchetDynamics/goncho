package shared

import "github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/ranking"

func TopN(values []string, n int) []string {
	return ranking.TopN(values, n)
}

package shared

import "github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared/collection"

func TopN(values []string, n int) []string {
	return collection.TopN(values, n)
}

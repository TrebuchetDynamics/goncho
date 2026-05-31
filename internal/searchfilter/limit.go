package searchfilter

import "github.com/TrebuchetDynamics/goncho/internal/searchfilter/policy"

func NormalizeLimit(limit int) int {
	return policy.NormalizeLimit(limit)
}

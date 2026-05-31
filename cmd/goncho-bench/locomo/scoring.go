package locomo

import "github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/locomo/scoring"

func RecallAny(retrieved, gold []string, k int) float64 {
	return scoring.RecallAny(retrieved, gold, k)
}

func StrictRecall(retrieved, gold []string, k int) float64 {
	return scoring.StrictRecall(retrieved, gold, k)
}

func NDCG(retrieved, gold []string, k int) float64 {
	return scoring.NDCG(retrieved, gold, k)
}

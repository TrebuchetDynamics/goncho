package scoring

import (
	"math"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/locomo/rankwindow"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/metrics"
)

func RecallAny(retrieved, gold []string, k int) float64 {
	seen := rankwindow.IDSet(retrieved, k)
	for _, id := range gold {
		if _, ok := seen[id]; ok {
			return 1
		}
	}
	return 0
}

func StrictRecall(retrieved, gold []string, k int) float64 {
	seen := rankwindow.IDSet(retrieved, k)
	for _, id := range gold {
		if _, ok := seen[id]; !ok {
			return 0
		}
	}
	return 1
}

func NDCG(retrieved, gold []string, k int) float64 {
	if k <= 0 || len(gold) == 0 {
		return 0
	}
	goldSet := map[string]struct{}{}
	for _, id := range gold {
		goldSet[id] = struct{}{}
	}
	seenRelevant := map[string]struct{}{}
	dcg := 0.0
	for i, id := range rankwindow.IDs(retrieved, k) {
		if _, ok := goldSet[id]; !ok {
			continue
		}
		if _, ok := seenRelevant[id]; ok {
			continue
		}
		seenRelevant[id] = struct{}{}
		dcg += 1 / math.Log2(float64(i+2))
	}
	idealCount := len(rankwindow.IDs(gold, k))
	idcg := 0.0
	for i := 0; i < idealCount; i++ {
		idcg += 1 / math.Log2(float64(i+2))
	}
	if idcg == 0 {
		return 0
	}
	return metrics.Round(dcg / idcg)
}

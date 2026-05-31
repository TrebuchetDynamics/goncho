package bootstrap

import (
	"math/rand"
	"sort"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/paired/comparisoncontract/verdict"
	sharedscore "github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared/score"
)

func MeanCI(values []float64, samples int, seed int64) verdict.CI {
	if len(values) == 0 || samples <= 0 {
		return verdict.CI{}
	}
	rng := rand.New(rand.NewSource(seed))
	means := make([]float64, 0, samples)
	for i := 0; i < samples; i++ {
		total := 0.0
		for range values {
			total += values[rng.Intn(len(values))]
		}
		means = append(means, total/float64(len(values)))
	}
	sort.Float64s(means)
	lowerIndex := int(0.025 * float64(samples))
	upperIndex := int(0.975 * float64(samples))
	if upperIndex >= len(means) {
		upperIndex = len(means) - 1
	}
	return verdict.CI{Lower: sharedscore.RoundSignedMetric(means[lowerIndex]), Upper: sharedscore.RoundSignedMetric(means[upperIndex])}
}

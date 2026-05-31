package comparisoncontract

import (
	"math/rand"
	"sort"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared"
)

type CI struct {
	Lower float64 `json:"lower"`
	Upper float64 `json:"upper"`
}

type ScoreSummary struct {
	PairedCount       int
	BaselineAvgScore  float64
	CandidateAvgScore float64
	ScoreDelta        float64
	BaselineWins      int
	CandidateWins     int
	Ties              int
	baselineTally     shared.ScoreTally
	candidateTally    shared.ScoreTally
}

func (s *ScoreSummary) Add(baselineScore, candidateScore float64, winner string) {
	s.PairedCount++
	s.baselineTally.Add(baselineScore)
	s.candidateTally.Add(candidateScore)
	s.BaselineAvgScore = s.baselineTally.Average()
	s.CandidateAvgScore = s.candidateTally.Average()
	s.ScoreDelta = shared.RoundSignedMetric(s.CandidateAvgScore - s.BaselineAvgScore)
	s.AddWinner(winner)
}

func (s *ScoreSummary) AddWinner(winner string) {
	switch winner {
	case "candidate":
		s.CandidateWins++
	case "baseline":
		s.BaselineWins++
	default:
		s.Ties++
	}
}

func Winner(baseScore, candidateScore float64) string {
	switch {
	case candidateScore > baseScore:
		return "candidate"
	case baseScore > candidateScore:
		return "baseline"
	default:
		return "tie"
	}
}

func Conclusion(ci CI, effectSizeFloor float64) (string, string) {
	if ci.Lower > effectSizeFloor {
		return "candidate_superior", "candidate_ci_above_effect_floor"
	}
	if ci.Upper < -effectSizeFloor {
		return "baseline_superior", "baseline_ci_below_negative_effect_floor"
	}
	return "inconclusive", "ci_overlaps_effect_floor"
}

func PointConclusion(delta, effectSizeFloor float64) (string, string) {
	if delta > effectSizeFloor {
		return "candidate_superior", "candidate_delta_above_effect_floor"
	}
	if delta < -effectSizeFloor {
		return "baseline_superior", "baseline_delta_below_negative_effect_floor"
	}
	return "inconclusive", "delta_within_effect_floor"
}

func BootstrapMeanCI(values []float64, samples int, seed int64) CI {
	if len(values) == 0 || samples <= 0 {
		return CI{}
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
	return CI{Lower: shared.RoundSignedMetric(means[lowerIndex]), Upper: shared.RoundSignedMetric(means[upperIndex])}
}

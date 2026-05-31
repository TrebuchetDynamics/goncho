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

type Report struct {
	GeneratedAt          string           `json:"generated_at"`
	SourcePath           string           `json:"source_path"`
	BaselineConfigID     string           `json:"baseline_config_id"`
	CandidateConfigID    string           `json:"candidate_config_id"`
	PairedCount          int              `json:"paired_count"`
	DroppedUnpairedCount int              `json:"dropped_unpaired_count"`
	BaselineAvgScore     float64          `json:"baseline_avg_score"`
	CandidateAvgScore    float64          `json:"candidate_avg_score"`
	ScoreDelta           float64          `json:"score_delta"`
	EffectSizeFloor      float64          `json:"effect_size_floor"`
	Conclusion           string           `json:"conclusion"`
	ConclusionReason     string           `json:"conclusion_reason"`
	BaselineWins         int              `json:"baseline_wins"`
	CandidateWins        int              `json:"candidate_wins"`
	Ties                 int              `json:"ties"`
	BootstrapSamples     int              `json:"bootstrap_samples"`
	BootstrapSeed        int64            `json:"bootstrap_seed"`
	ScoreDeltaCI95       CI               `json:"score_delta_ci95"`
	ByAbility            map[string]Stats `json:"by_ability"`
	Rows                 []Row            `json:"rows"`
}

type Stats struct {
	PairedCount       int     `json:"paired_count"`
	BaselineAvgScore  float64 `json:"baseline_avg_score"`
	CandidateAvgScore float64 `json:"candidate_avg_score"`
	ScoreDelta        float64 `json:"score_delta"`
	Conclusion        string  `json:"conclusion"`
	ConclusionReason  string  `json:"conclusion_reason"`
	BaselineWins      int     `json:"baseline_wins"`
	CandidateWins     int     `json:"candidate_wins"`
	Ties              int     `json:"ties"`
}

type Row struct {
	Scale                 string  `json:"scale"`
	ConversationID        string  `json:"conversation_id"`
	QID                   string  `json:"qid"`
	BaselineQID           string  `json:"baseline_qid,omitempty"`
	CandidateQID          string  `json:"candidate_qid,omitempty"`
	BaselineSourcePath    string  `json:"baseline_source_path,omitempty"`
	BaselineSourceSHA256  string  `json:"baseline_source_sha256,omitempty"`
	CandidateSourcePath   string  `json:"candidate_source_path,omitempty"`
	CandidateSourceSHA256 string  `json:"candidate_source_sha256,omitempty"`
	MatchKey              string  `json:"match_key"`
	Ability               string  `json:"ability"`
	Question              string  `json:"question,omitempty"`
	BaselineScore         float64 `json:"baseline_score"`
	CandidateScore        float64 `json:"candidate_score"`
	ScoreDelta            float64 `json:"score_delta"`
	BaselineCorrect       bool    `json:"baseline_correct"`
	CandidateCorrect      bool    `json:"candidate_correct"`
	Winner                string  `json:"winner"`
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

func SummarizeRows(rows []Row, bootstrapSamples int, bootstrapSeed int64, effectSizeFloor float64) Report {
	report := Report{
		PairedCount:      len(rows),
		EffectSizeFloor:  shared.RoundMetric(effectSizeFloor),
		BootstrapSamples: bootstrapSamples,
		BootstrapSeed:    bootstrapSeed,
		ByAbility:        map[string]Stats{},
		Rows:             append([]Row(nil), rows...),
	}
	abilityRows := map[string][]Row{}
	var summary ScoreSummary
	diffs := make([]float64, 0, len(rows))
	for _, row := range rows {
		summary.Add(row.BaselineScore, row.CandidateScore, row.Winner)
		diffs = append(diffs, row.CandidateScore-row.BaselineScore)
		abilityRows[row.Ability] = append(abilityRows[row.Ability], row)
	}
	report.BaselineAvgScore = summary.BaselineAvgScore
	report.CandidateAvgScore = summary.CandidateAvgScore
	report.ScoreDelta = summary.ScoreDelta
	report.CandidateWins = summary.CandidateWins
	report.BaselineWins = summary.BaselineWins
	report.Ties = summary.Ties
	report.ScoreDeltaCI95 = BootstrapMeanCI(diffs, bootstrapSamples, bootstrapSeed)
	report.Conclusion, report.ConclusionReason = Conclusion(report.ScoreDeltaCI95, report.EffectSizeFloor)
	for ability, rows := range abilityRows {
		report.ByAbility[ability] = StatsForRows(rows, report.EffectSizeFloor)
	}
	return report
}

func StatsForRows(rows []Row, effectSizeFloor float64) Stats {
	var summary ScoreSummary
	for _, row := range rows {
		summary.Add(row.BaselineScore, row.CandidateScore, row.Winner)
	}
	stats := Stats{
		PairedCount:       summary.PairedCount,
		BaselineAvgScore:  summary.BaselineAvgScore,
		CandidateAvgScore: summary.CandidateAvgScore,
		ScoreDelta:        summary.ScoreDelta,
		BaselineWins:      summary.BaselineWins,
		CandidateWins:     summary.CandidateWins,
		Ties:              summary.Ties,
	}
	stats.Conclusion, stats.ConclusionReason = PointConclusion(stats.ScoreDelta, effectSizeFloor)
	return stats
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

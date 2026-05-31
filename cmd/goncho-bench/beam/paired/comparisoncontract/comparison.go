package comparisoncontract

import "github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared"

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

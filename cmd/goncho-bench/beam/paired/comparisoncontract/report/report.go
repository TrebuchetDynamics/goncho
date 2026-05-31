package report

import "github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/paired/comparisoncontract/verdict"

// Report is the JSON contract for a deterministic paired outcome comparison.
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
	ScoreDeltaCI95       verdict.CI       `json:"score_delta_ci95"`
	ByAbility            map[string]Stats `json:"by_ability"`
	Rows                 []Row            `json:"rows"`
}

// Stats is the JSON contract for aggregate score comparison statistics.
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

// Row is the JSON contract for a single paired baseline/candidate outcome.
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

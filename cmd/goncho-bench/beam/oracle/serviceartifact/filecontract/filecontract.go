package filecontract

import (
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/artifactcontract"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared"
)

// SummaryFile is the stable JSON contract for BEAM service summary artifacts.
type SummaryFile struct {
	Date           string                             `json:"date"`
	Metadata       SummaryMetadata                    `json:"metadata"`
	AbilitySummary map[string]map[string]AbilityStats `json:"ability_summary"`
}

// SummaryMetadata describes a BEAM service summary artifact run.
type SummaryMetadata struct {
	Model       string `json:"model"`
	SampleSize  int    `json:"sample_size"`
	JudgeModel  string `json:"judge_model"`
	ConfigID    string `json:"config_id"`
	PureRecall  bool   `json:"pure_recall"`
	Service     string `json:"service"`
	Corpus      string `json:"corpus_version"`
	CaseCount   int    `json:"case_count"`
	Description string `json:"description"`
}

// AbilityStats summarizes scores for one ability bucket.
type AbilityStats struct {
	AvgScore float64 `json:"avg_score"`
	Count    int     `json:"count"`
}

// PairedOutcome reuses the shared paired-comparison artifact row contract.
type PairedOutcome = shared.PairedOutcome

// FailureAuditRow is the stable JSONL contract for failed BEAM service cases.
type FailureAuditRow struct {
	ConfigID              string   `json:"config_id"`
	RunStartedAt          string   `json:"run_started_at"`
	Scale                 string   `json:"scale"`
	ConversationID        string   `json:"conversation_id"`
	QID                   string   `json:"qid"`
	Ability               string   `json:"ability"`
	Question              string   `json:"question"`
	Score                 float64  `json:"score"`
	FailureMode           string   `json:"failure_mode"`
	Rank                  int      `json:"rank"`
	RelevantIDs           []string `json:"relevant_ids"`
	RequiredEvidenceKinds []string `json:"required_evidence_kinds,omitempty"`
	ExpectedNoAnswer      bool     `json:"expected_no_answer,omitempty"`
	CandidateMemoryIDs    []string `json:"candidate_memory_ids"`
	SelectedMemoryIDs     []string `json:"selected_memory_ids"`
	RetrievedTop10        []string `json:"retrieved_top_10"`
	SelectedEvidenceKinds []string `json:"selected_evidence_kinds,omitempty"`
	TopEvidenceKinds      []string `json:"top_evidence_kinds,omitempty"`
	RecallAt5             float64  `json:"recall_at_5"`
	RecallAt10            float64  `json:"recall_at_10"`
	ContextSatisfied      bool     `json:"context_satisfied"`
	ProvenanceSatisfied   bool     `json:"provenance_satisfied"`
	TokenBudgetWithin     bool     `json:"token_budget_within"`
	WarningCodes          []string `json:"warning_codes,omitempty"`
}

// ResultsFile is the stable JSON contract for BEAM service result artifacts.
type ResultsFile struct {
	Metadata ResultsMetadata       `json:"metadata"`
	Results  []ConversationResults `json:"results"`
}

// ResultsMetadata describes a BEAM service results artifact run.
type ResultsMetadata struct {
	Date               string                 `json:"date"`
	RunStartedAt       string                 `json:"run_started_at"`
	ConfigID           string                 `json:"config_id"`
	Model              string                 `json:"model"`
	JudgeModel         string                 `json:"judge_model"`
	TopK               int                    `json:"top_k"`
	SampleSize         int                    `json:"sample_size"`
	Scales             []string               `json:"scales"`
	TotalConversations int                    `json:"total_conversations"`
	PureRecall         bool                   `json:"pure_recall"`
	Config             map[string]any         `json:"config"`
	Diagnostics        map[string]interface{} `json:"diagnostics"`
}

// ConversationResults groups question results for one BEAM conversation.
type ConversationResults struct {
	ConversationID string           `json:"conversation_id"`
	Scale          string           `json:"scale"`
	NumQuestions   int              `json:"num_questions"`
	NumEvaluated   int              `json:"num_evaluated"`
	Results        []QuestionResult `json:"results"`
}

// QuestionResult is one stable BEAM service result row.
type QuestionResult struct {
	QID                  string                            `json:"qid"`
	Ability              string                            `json:"ability"`
	Question             string                            `json:"question,omitempty"`
	IdealAnswer          string                            `json:"ideal_answer,omitempty"`
	Rubric               []string                          `json:"rubric,omitempty"`
	RubricContextScore   float64                           `json:"rubric_context_score,omitempty"`
	RubricContextMatches []string                          `json:"rubric_context_matches,omitempty"`
	AIAnswer             string                            `json:"ai_answer"`
	RecallProvenance     artifactcontract.RecallProvenance `json:"recall_provenance"`
	Score                float64                           `json:"score"`
	Nuggets              []string                          `json:"nuggets"`
	Assessment           string                            `json:"assessment"`
	AnswerTimeMS         float64                           `json:"answer_time_ms"`
	JudgeTimeMS          float64                           `json:"judge_time_ms"`
}

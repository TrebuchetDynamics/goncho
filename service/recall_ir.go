package goncho

import (
	"encoding/json"
	"time"
)

const (
	RecallStageGenerate = "generate"
	RecallStageScore    = "score"
	RecallStageSelect   = "select"
	RecallStageProject  = "project"

	RecallWarningInfo     = "info"
	RecallWarningDegraded = "degraded"
	RecallWarningError    = "error"

	RecallWarningSemanticUnavailable        = "semantic_unavailable"
	RecallWarningGraphDisabled              = "graph_disabled"
	RecallWarningStaleEmbeddingIndex        = "stale_embedding_index"
	RecallWarningFTSUnavailable             = "fts_unavailable"
	RecallWarningScopeExcludedAllCandidates = "scope_excluded_all_candidates"
	RecallWarningTokenBudgetTruncated       = "token_budget_truncated"
	RecallWarningSupersededEvidenceObserved = "superseded_evidence_observed"

	RecallRejectScopeMismatch = "scope_mismatch"
	RecallRejectTokenBudget   = "token_budget"
	RecallRejectNotSelected   = "not_selected"
)

type RecallQuery struct {
	WorkspaceID string   `json:"workspace_id"`
	Peer        string   `json:"peer"`
	Query       string   `json:"query"`
	SessionKey  string   `json:"session_key,omitempty"`
	ScopeID     string   `json:"scope_id,omitempty"`
	Sources     []string `json:"sources,omitempty"`
	Limit       int      `json:"limit,omitempty"`
	MaxTokens   int      `json:"max_tokens,omitempty"`
}

type EvidenceItem struct {
	Kind     string            `json:"kind"`
	Source   string            `json:"source,omitempty"`
	ID       string            `json:"id,omitempty"`
	Note     string            `json:"note,omitempty"`
	Score    float64           `json:"score,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type RecallCandidate struct {
	MemoryID   string         `json:"memory_id"`
	SourceType string         `json:"source_type"`
	Content    string         `json:"content"`
	SessionID  string         `json:"session_id,omitempty"`
	AgentID    string         `json:"agent_id,omitempty"`
	ScopeID    string         `json:"scope_id,omitempty"`
	CreatedAt  time.Time      `json:"created_at,omitempty"`
	Importance float64        `json:"importance,omitempty"`
	Provenance []EvidenceItem `json:"provenance,omitempty"`
}

type RecallScore struct {
	KeywordScore     float64  `json:"keyword_score"`
	SemanticScore    float64  `json:"semantic_score"`
	GraphScore       float64  `json:"graph_score"`
	FactScore        float64  `json:"fact_score,omitempty"`
	RecencyScore     float64  `json:"recency_score"`
	ImportanceScore  float64  `json:"importance_score"`
	ScopeScore       float64  `json:"scope_score"`
	RRFScore         float64  `json:"rrf_score"`
	DiversityPenalty float64  `json:"diversity_penalty"`
	FinalScore       float64  `json:"final_score"`
	WhySelected      []string `json:"why_selected,omitempty"`
}

type ScoredRecallCandidate struct {
	Candidate RecallCandidate `json:"candidate"`
	Score     RecallScore     `json:"score"`
}

type RejectedRecallCandidate struct {
	Candidate   RecallCandidate `json:"candidate"`
	Score       RecallScore     `json:"score"`
	Reason      string          `json:"reason"`
	WhyRejected []string        `json:"why_rejected,omitempty"`
}

type RecallScoringConfig struct {
	Version       string             `json:"version"`
	Weights       map[string]float64 `json:"weights"`
	RRFK          int                `json:"rrf_k"`
	MMRLambda     float64            `json:"mmr_lambda"`
	DiversityKeys []string           `json:"diversity_keys,omitempty"`
	TokenBudget   int                `json:"token_budget,omitempty"`
}

type RecallWarning struct {
	Code     string            `json:"code"`
	Stage    string            `json:"stage"`
	Severity string            `json:"severity"`
	Message  string            `json:"message,omitempty"`
	Evidence map[string]string `json:"evidence,omitempty"`
}

type RecallVoiceDiagnostic struct {
	Name           string  `json:"name"`
	Enabled        bool    `json:"enabled"`
	Weight         float64 `json:"weight"`
	CandidatesWith int     `json:"candidates_with"`
	MaxScore       float64 `json:"max_score"`
	MinScore       float64 `json:"min_score"`
	AvgScore       float64 `json:"avg_score"`
	SelectedCount  int     `json:"selected_count"`
}

type RecallTrace struct {
	TraceID         string                    `json:"trace_id"`
	PipelineVersion string                    `json:"pipeline_version"`
	CreatedAt       time.Time                 `json:"created_at"`
	Query           RecallQuery               `json:"query"`
	ScoringConfig   RecallScoringConfig       `json:"scoring_config"`
	VoiceDiagnostics []RecallVoiceDiagnostic  `json:"voice_diagnostics,omitempty"`
	Candidates      []ScoredRecallCandidate   `json:"candidates"`
	Selected        []ScoredRecallCandidate   `json:"selected"`
	Rejected        []RejectedRecallCandidate `json:"rejected"`
	Warnings        []RecallWarning           `json:"warnings"`
}

func (t RecallTrace) StableJSON() ([]byte, error) {
	normalized := normalizeRecallTrace(t)
	raw, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return nil, err
	}
	raw = append(raw, '\n')
	return raw, nil
}

func normalizeRecallTrace(trace RecallTrace) RecallTrace {
	if trace.Candidates == nil {
		trace.Candidates = []ScoredRecallCandidate{}
	}
	if trace.Selected == nil {
		trace.Selected = []ScoredRecallCandidate{}
	}
	if trace.Rejected == nil {
		trace.Rejected = []RejectedRecallCandidate{}
	}
	if trace.Warnings == nil {
		trace.Warnings = []RecallWarning{}
	}
	if trace.ScoringConfig.Weights == nil {
		trace.ScoringConfig.Weights = map[string]float64{}
	}
	return trace
}

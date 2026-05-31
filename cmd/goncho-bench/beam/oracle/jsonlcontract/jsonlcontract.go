package jsonlcontract

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/casecontract"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared"
	goncho "github.com/TrebuchetDynamics/goncho/service"
)

type Record struct {
	Type                  string   `json:"type"`
	Dataset               string   `json:"dataset,omitempty"`
	Scale                 string   `json:"scale,omitempty"`
	ID                    string   `json:"id,omitempty"`
	ConversationID        string   `json:"conversation_id,omitempty"`
	Peer                  string   `json:"peer,omitempty"`
	SessionKey            string   `json:"session_key,omitempty"`
	Content               string   `json:"content,omitempty"`
	Ability               string   `json:"ability,omitempty"`
	Query                 string   `json:"query,omitempty"`
	IdealAnswer           string   `json:"ideal_answer,omitempty"`
	Rubric                []string `json:"rubric,omitempty"`
	RelevantIDs           []string `json:"relevant_ids,omitempty"`
	ContextContains       []string `json:"context_contains,omitempty"`
	RequiredEvidenceKinds []string `json:"required_evidence_kinds,omitempty"`
	ExpectedNoAnswer      bool     `json:"expected_no_answer,omitempty"`
	Limit                 int      `json:"limit,omitempty"`
	MaxTokens             int      `json:"max_tokens,omitempty"`
}

type Question struct {
	ID                    string
	Scale                 string
	ConversationID        string
	Peer                  string
	SessionKey            string
	Ability               string
	Query                 string
	IdealAnswer           string
	Rubric                []string
	RelevantIDs           []string
	ContextContains       []string
	RequiredEvidenceKinds []string
	ExpectedNoAnswer      bool
	Limit                 int
	MaxTokens             int
}

func NormalizeConversationID(conversationID string) string {
	return shared.FirstNonEmptyTrimmed(conversationID, casecontract.DefaultConversationID)
}

func ScoringConfig(question Question) goncho.RecallScoringConfig {
	version := "beam-jsonl-" + strings.ToLower(strings.TrimSpace(question.Ability)) + "-v1"
	for _, kind := range question.RequiredEvidenceKinds {
		if shared.NormalizeEvidenceKind(kind) == "graph" {
			return goncho.RecallScoringConfig{
				Version:     version,
				Weights:     map[string]float64{"keyword": 0.05, "fact": 0.10, "graph": 0.80, "scope": 0.05},
				RRFK:        60,
				MMRLambda:   1,
				TokenBudget: 320,
			}
		}
	}
	return goncho.RecallScoringConfig{
		Version:     version,
		Weights:     map[string]float64{"keyword": 0.10, "fact": 0.75, "graph": 0.05, "scope": 0.10},
		RRFK:        60,
		MMRLambda:   1,
		TokenBudget: 320,
	}
}

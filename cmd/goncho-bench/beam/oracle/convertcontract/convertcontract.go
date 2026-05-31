package convertcontract

import "encoding/json"

// HuggingFaceRecord is one source BEAM HuggingFace JSONL row.
type HuggingFaceRecord struct {
	ConversationID string          `json:"conversation_id"`
	Scale          string          `json:"scale"`
	Chat           json.RawMessage `json:"chat"`
	Plans          json.RawMessage `json:"plans"`
	Questions      json.RawMessage `json:"probing_questions"`
}

// Message is a normalized conversation message produced by the converter.
type Message struct {
	Role    string
	Content string
}

// Question is a normalized BEAM probing question produced by the converter.
type Question struct {
	ID                    string   `json:"id"`
	QID                   string   `json:"qid"`
	Question              string   `json:"question"`
	Prompt                string   `json:"prompt"`
	Query                 string   `json:"query"`
	IdealAnswer           string   `json:"ideal_answer"`
	IdealResponse         string   `json:"ideal_response"`
	Answer                string   `json:"answer"`
	IdealSummary          string   `json:"ideal_summary"`
	Rubric                []string `json:"rubric"`
	RelevantIDs           []string `json:"relevant_ids"`
	RelevantMessageIdxs   []int    `json:"relevant_message_indices"`
	EvidenceMessageIdxs   []int    `json:"evidence_message_indices"`
	SourceMessageIdxs     []int    `json:"source_message_indices"`
	RequiredEvidenceKinds []string `json:"required_evidence_kinds"`
	ExpectedNoAnswer      bool     `json:"expected_no_answer"`
	Limit                 int      `json:"limit"`
	MaxTokens             int      `json:"max_tokens"`
}

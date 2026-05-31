package shared

// PairedOutcome is the JSONL contract used by BEAM oracle runs and paired comparisons.
type PairedOutcome struct {
	ConfigID       string  `json:"config_id"`
	RunStartedAt   string  `json:"run_started_at"`
	Scale          string  `json:"scale"`
	ConversationID string  `json:"conversation_id"`
	QID            string  `json:"qid"`
	Ability        string  `json:"ability"`
	Question       string  `json:"question,omitempty"`
	SourcePath     string  `json:"source_path,omitempty"`
	SourceSHA256   string  `json:"source_sha256,omitempty"`
	Score          float64 `json:"score"`
	Correct        bool    `json:"correct"`
}

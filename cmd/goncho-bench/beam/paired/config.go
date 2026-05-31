package paired

type Config struct {
	ComparePath             string
	BaselineConfigID        string
	CandidateConfigID       string
	CompareJSONOut          string
	CompareMarkdownOut      string
	CompareBootstrapSamples int
	CompareEffectSizeFloor  float64
	ResultsIn               string
	ResultsOut              string
	ResultsConfigID         string
}

type servicePairedOutcome struct {
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

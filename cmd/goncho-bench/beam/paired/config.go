package paired

import (
	"crypto/sha256"
	"encoding/hex"
	"math"
)

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

func roundMetric(v float64) float64 {
	return math.Round(v*10000) / 10000
}

func checksumBytesSHA256(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

package artifactcontract

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/casecontract"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared"
	goncho "github.com/TrebuchetDynamics/goncho/service"
)

// CaseFields is the shared trimmed/canonical case projection used by BEAM artifact rows.
type CaseFields struct {
	Scale                string   `json:"scale"`
	ConversationID       string   `json:"conversation_id"`
	QID                  string   `json:"qid"`
	Ability              string   `json:"ability"`
	Question             string   `json:"question"`
	CandidateMemoryIDs   []string `json:"candidate_memory_ids,omitempty"`
	SelectedMemoryIDs    []string `json:"selected_memory_ids,omitempty"`
	RubricContextScore   float64  `json:"rubric_context_score,omitempty"`
	RubricContextMatches []string `json:"rubric_context_matches,omitempty"`
}

// BuildCaseFields projects a benchmark case into shared canonical artifact fields.
func BuildCaseFields(c goncho.RecallBenchmarkCaseReport) CaseFields {
	return CaseFields{
		Scale:                casecontract.Scale(c),
		ConversationID:       casecontract.ConversationID(c),
		QID:                  c.ID,
		Ability:              shared.NormalizeAbility(c.Ability),
		Question:             strings.TrimSpace(c.Question),
		CandidateMemoryIDs:   append([]string(nil), c.CandidateMemoryIDs...),
		SelectedMemoryIDs:    append([]string(nil), c.SelectedMemoryIDs...),
		RubricContextScore:   c.RubricContextScore,
		RubricContextMatches: append([]string(nil), c.RubricContextMatches...),
	}
}

// RecallProvenance is the shared artifact shape for BEAM service result and judge-request rows.
type RecallProvenance struct {
	Engine             string             `json:"engine"`
	KeptCount          int                `json:"kept_count"`
	VoiceSums          map[string]float64 `json:"voice_sums"`
	TopResultVoices    map[string]float64 `json:"top_result_voices"`
	TopResultTier      string             `json:"top_result_tier"`
	CandidateMemoryIDs []string           `json:"candidate_memory_ids,omitempty"`
	SelectedMemoryIDs  []string           `json:"selected_memory_ids,omitempty"`
}

// BuildRecallProvenance projects a benchmark case into the stable artifact provenance contract.
func BuildRecallProvenance(c goncho.RecallBenchmarkCaseReport) RecallProvenance {
	return RecallProvenance{
		Engine:             casecontract.ModelName,
		KeptCount:          len(c.CandidateMemoryIDs),
		VoiceSums:          casecontract.VoiceMap(c.SelectedEvidenceKinds),
		TopResultVoices:    casecontract.VoiceMap(c.TopEvidenceKinds),
		TopResultTier:      casecontract.TopResultTier(c.TopEvidenceKinds),
		CandidateMemoryIDs: append([]string(nil), c.CandidateMemoryIDs...),
		SelectedMemoryIDs:  append([]string(nil), c.SelectedMemoryIDs...),
	}
}

package artifactcontract

import (
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/casecontract"
	goncho "github.com/TrebuchetDynamics/goncho/service"
)

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

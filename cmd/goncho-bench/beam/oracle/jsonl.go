package oracle

import (
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/jsonlcontract"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared"
	"github.com/TrebuchetDynamics/goncho/service"
)

type beamJSONLRecord = jsonlcontract.Record

type beamJSONLQuestion = jsonlcontract.Question

func loadBeamServiceJSONLCases(path string) ([]goncho.RecallBenchmarkServiceCase, error) {
	records, err := shared.ReadJSONLFile[beamJSONLRecord](path, "goncho-bench: open BEAM JSONL dataset", "goncho-bench: read BEAM JSONL dataset", "goncho-bench: decode BEAM JSONL")
	if err != nil {
		return nil, err
	}
	return beamServiceCasesFromJSONLRecords(records)
}

func beamServiceCasesFromJSONLRecords(records []beamJSONLRecord) ([]goncho.RecallBenchmarkServiceCase, error) {
	return jsonlcontract.ServiceCasesFromRecords(records, beamServiceScale)
}

func beamJSONLMemory(record beamJSONLRecord, lineNo int) (goncho.RecallBenchmarkServiceMemory, string, error) {
	return jsonlcontract.MemoryFromRecord(record, lineNo)
}

func beamJSONLQuestionFromRecord(record beamJSONLRecord, defaultScale string, lineNo int) (beamJSONLQuestion, error) {
	return jsonlcontract.QuestionFromRecord(record, defaultScale, lineNo)
}

func normalizeBeamJSONLConversationID(conversationID string) string {
	return jsonlcontract.NormalizeConversationID(conversationID)
}

func beamJSONLScoringConfig(question beamJSONLQuestion) goncho.RecallScoringConfig {
	return jsonlcontract.ScoringConfig(question)
}

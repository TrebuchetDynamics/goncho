package oracle

import (
	"fmt"
	"strings"

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
	defaultScale := beamServiceScale
	memoriesByConversation := map[string][]goncho.RecallBenchmarkServiceMemory{}
	questions := []beamJSONLQuestion{}
	for i, record := range records {
		lineNo := i + 1
		switch shared.NormalizeRecordType(record.Type) {
		case "meta":
			if scale := strings.TrimSpace(record.Scale); scale != "" {
				defaultScale = scale
			}
		case "memory":
			memory, conversationID, err := beamJSONLMemory(record, lineNo)
			if err != nil {
				return nil, err
			}
			memoriesByConversation[conversationID] = append(memoriesByConversation[conversationID], memory)
		case "question":
			question, err := beamJSONLQuestionFromRecord(record, defaultScale, lineNo)
			if err != nil {
				return nil, err
			}
			questions = append(questions, question)
		default:
			return nil, fmt.Errorf("goncho-bench: BEAM JSONL line %d has unknown type %q", lineNo, record.Type)
		}
	}
	if len(questions) == 0 {
		return nil, fmt.Errorf("goncho-bench: BEAM JSONL dataset has no question records")
	}

	cases := make([]goncho.RecallBenchmarkServiceCase, 0, len(questions))
	for _, question := range questions {
		memories := memoriesByConversation[question.ConversationID]
		if len(memories) == 0 && !question.ExpectedNoAnswer {
			return nil, fmt.Errorf("goncho-bench: BEAM question %q references conversation %q with no memories", question.ID, question.ConversationID)
		}
		cases = append(cases, goncho.RecallBenchmarkServiceCase{
			ID:                    question.ID,
			Ability:               question.Ability,
			Scale:                 question.Scale,
			ConversationID:        question.ConversationID,
			Peer:                  question.Peer,
			SessionKey:            question.SessionKey,
			Query:                 question.Query,
			IdealAnswer:           question.IdealAnswer,
			Rubric:                append([]string(nil), question.Rubric...),
			Memories:              append([]goncho.RecallBenchmarkServiceMemory(nil), memories...),
			RelevantRefs:          append([]string(nil), question.RelevantIDs...),
			ContextContains:       append([]string(nil), question.ContextContains...),
			RequiredEvidenceKinds: append([]string(nil), question.RequiredEvidenceKinds...),
			ExpectedNoAnswer:      question.ExpectedNoAnswer,
			Limit:                 question.Limit,
			MaxTokens:             question.MaxTokens,
			ScoringConfig:         beamJSONLScoringConfig(question),
		})
	}
	return cases, nil
}

func beamJSONLMemory(record beamJSONLRecord, lineNo int) (goncho.RecallBenchmarkServiceMemory, string, error) {
	id := strings.TrimSpace(record.ID)
	if id == "" {
		return goncho.RecallBenchmarkServiceMemory{}, "", fmt.Errorf("goncho-bench: BEAM memory line %d missing id", lineNo)
	}
	conversationID := normalizeBeamJSONLConversationID(record.ConversationID)
	content := strings.TrimSpace(record.Content)
	if content == "" {
		return goncho.RecallBenchmarkServiceMemory{}, "", fmt.Errorf("goncho-bench: BEAM memory %q missing content", id)
	}
	return goncho.RecallBenchmarkServiceMemory{
		Ref:        id,
		Conclusion: content,
		Peer:       strings.TrimSpace(record.Peer),
		SessionKey: strings.TrimSpace(record.SessionKey),
	}, conversationID, nil
}

func beamJSONLQuestionFromRecord(record beamJSONLRecord, defaultScale string, lineNo int) (beamJSONLQuestion, error) {
	id := strings.TrimSpace(record.ID)
	if id == "" {
		return beamJSONLQuestion{}, fmt.Errorf("goncho-bench: BEAM question line %d missing id", lineNo)
	}
	query := strings.TrimSpace(record.Query)
	if query == "" {
		return beamJSONLQuestion{}, fmt.Errorf("goncho-bench: BEAM question %q missing query", id)
	}
	scale := shared.FirstNonEmptyTrimmed(record.Scale, defaultScale, beamServiceScale)
	return beamJSONLQuestion{
		ID:                    id,
		Scale:                 scale,
		ConversationID:        normalizeBeamJSONLConversationID(record.ConversationID),
		Peer:                  strings.TrimSpace(record.Peer),
		SessionKey:            strings.TrimSpace(record.SessionKey),
		Ability:               shared.NormalizeAbility(record.Ability),
		Query:                 query,
		IdealAnswer:           strings.TrimSpace(record.IdealAnswer),
		Rubric:                append([]string(nil), record.Rubric...),
		RelevantIDs:           append([]string(nil), record.RelevantIDs...),
		ContextContains:       append([]string(nil), record.ContextContains...),
		RequiredEvidenceKinds: append([]string(nil), record.RequiredEvidenceKinds...),
		ExpectedNoAnswer:      record.ExpectedNoAnswer,
		Limit:                 record.Limit,
		MaxTokens:             record.MaxTokens,
	}, nil
}

func normalizeBeamJSONLConversationID(conversationID string) string {
	return jsonlcontract.NormalizeConversationID(conversationID)
}

func beamJSONLScoringConfig(question beamJSONLQuestion) goncho.RecallScoringConfig {
	return jsonlcontract.ScoringConfig(question)
}

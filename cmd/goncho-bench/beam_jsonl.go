package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/TrebuchetDynamics/goncho"
)

type beamJSONLRecord struct {
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

type beamJSONLQuestion struct {
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

func loadBeamServiceJSONLCases(path string) ([]goncho.RecallBenchmarkServiceCase, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("goncho-bench: open BEAM JSONL dataset: %w", err)
	}
	defer file.Close()

	records := []beamJSONLRecord{}
	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record beamJSONLRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			return nil, fmt.Errorf("goncho-bench: decode BEAM JSONL line %d: %w", lineNo, err)
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("goncho-bench: read BEAM JSONL dataset: %w", err)
	}
	return beamServiceCasesFromJSONLRecords(records)
}

func beamServiceCasesFromJSONLRecords(records []beamJSONLRecord) ([]goncho.RecallBenchmarkServiceCase, error) {
	defaultScale := beamServiceScale
	memoriesByConversation := map[string][]goncho.RecallBenchmarkServiceMemory{}
	questions := []beamJSONLQuestion{}
	for i, record := range records {
		lineNo := i + 1
		switch strings.ToLower(strings.TrimSpace(record.Type)) {
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
	scale := strings.TrimSpace(record.Scale)
	if scale == "" {
		scale = strings.TrimSpace(defaultScale)
	}
	if scale == "" {
		scale = beamServiceScale
	}
	return beamJSONLQuestion{
		ID:                    id,
		Scale:                 scale,
		ConversationID:        normalizeBeamJSONLConversationID(record.ConversationID),
		Peer:                  strings.TrimSpace(record.Peer),
		SessionKey:            strings.TrimSpace(record.SessionKey),
		Ability:               strings.ToUpper(strings.TrimSpace(record.Ability)),
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
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return beamServiceConversationID
	}
	return conversationID
}

func beamJSONLScoringConfig(question beamJSONLQuestion) goncho.RecallScoringConfig {
	version := "beam-jsonl-" + strings.ToLower(strings.TrimSpace(question.Ability)) + "-v1"
	for _, kind := range question.RequiredEvidenceKinds {
		if strings.EqualFold(strings.TrimSpace(kind), "graph") {
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

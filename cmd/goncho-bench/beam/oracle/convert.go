package oracle

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/convertcontract"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/convertpolicy"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared"
	"github.com/TrebuchetDynamics/goncho/internal/stringutil"
)

const beamConvertDefaultPeer = convertpolicy.DefaultPeer

type beamHuggingFaceRecord = convertcontract.HuggingFaceRecord

type beamConvertedMessage = convertcontract.Message

type beamConvertedQuestion = convertcontract.Question

type beamConversionDiagnostics = convertpolicy.Diagnostics

func ConvertHuggingFaceJSONL(inputPath, outputPath, fallbackScale string) error {
	inputPath = strings.TrimSpace(inputPath)
	if inputPath == "" {
		return fmt.Errorf("goncho-bench: --beam-convert-in is required")
	}
	outputPath = strings.TrimSpace(outputPath)
	if outputPath == "" {
		return fmt.Errorf("goncho-bench: --beam-convert-out is required for --beam-convert-in")
	}
	records, err := loadBeamHuggingFaceRecords(inputPath, fallbackScale)
	if err != nil {
		return err
	}
	return writeConvertedBeamJSONL(outputPath, records)
}

func loadBeamHuggingFaceRecords(path, fallbackScale string) ([]beamJSONLRecord, error) {
	records, _, err := loadBeamHuggingFaceRecordsWithDiagnostics(path, fallbackScale)
	return records, err
}

func loadBeamHuggingFaceRecordsWithDiagnostics(path, fallbackScale string) ([]beamJSONLRecord, beamConversionDiagnostics, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, beamConversionDiagnostics{}, fmt.Errorf("goncho-bench: open HuggingFace BEAM JSONL: %w", err)
	}
	defer file.Close()
	sourceHasher := sha256.New()

	fallbackScale = shared.FirstNonEmptyTrimmed(fallbackScale, beamServiceScale)
	out := []beamJSONLRecord{{Type: "meta", Dataset: "beam-huggingface-converted", Scale: fallbackScale}}
	scanner := shared.NewJSONLScanner(io.TeeReader(file, sourceHasher))
	if err := shared.ForEachNonEmptyJSONLLine(scanner, "goncho-bench: read HuggingFace BEAM JSONL", func(lineNo int, line string) error {
		var record beamHuggingFaceRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			return fmt.Errorf("goncho-bench: decode HuggingFace BEAM line %d: %w", lineNo, err)
		}
		converted, err := convertBeamHuggingFaceRecord(record, lineNo, fallbackScale)
		if err != nil {
			return err
		}
		out = append(out, converted...)
		return nil
	}); err != nil {
		return nil, beamConversionDiagnostics{}, err
	}
	if len(out) == 1 {
		return nil, beamConversionDiagnostics{}, fmt.Errorf("goncho-bench: HuggingFace BEAM JSONL has no conversation records")
	}
	diagnostics := summarizeBeamConversionRecords(out)
	diagnostics.SourceSHA256 = hex.EncodeToString(sourceHasher.Sum(nil))
	convertedRaw, err := encodeBeamJSONLBytes(out)
	if err != nil {
		return nil, beamConversionDiagnostics{}, err
	}
	diagnostics.ConvertedJSONLSHA256 = shared.ChecksumBytesSHA256(convertedRaw)
	return out, diagnostics, nil
}

func convertBeamHuggingFaceRecord(record beamHuggingFaceRecord, lineNo int, fallbackScale string) ([]beamJSONLRecord, error) {
	conversationID := shared.FirstNonEmptyTrimmed(record.ConversationID, fmt.Sprintf("beam-conversation-%06d", lineNo))
	scale := shared.FirstNonEmptyTrimmed(record.Scale, fallbackScale)
	messages, err := beamHuggingFaceMessages(record)
	if err != nil {
		return nil, fmt.Errorf("goncho-bench: convert BEAM conversation %q messages: %w", conversationID, err)
	}
	questionsByAbility, err := parseBeamHuggingFaceQuestions(record.Questions)
	if err != nil {
		return nil, fmt.Errorf("goncho-bench: convert BEAM conversation %q questions: %w", conversationID, err)
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("goncho-bench: BEAM conversation %q has no chat messages", conversationID)
	}
	messageIDs := make([]string, len(messages))
	out := make([]beamJSONLRecord, 0, len(messages)+beamQuestionCount(questionsByAbility))
	peer := beamConvertDefaultPeer
	sessionKey := conversationID
	idPrefix := stableBeamIDSegment(conversationID)
	for i, msg := range messages {
		memoryID := fmt.Sprintf("%s-mem-%06d", idPrefix, i+1)
		messageIDs[i] = memoryID
		content := strings.TrimSpace(msg.Content)
		role := strings.TrimSpace(msg.Role)
		if role != "" {
			content = role + ": " + content
		}
		out = append(out, beamJSONLRecord{
			Type:           "memory",
			ID:             memoryID,
			ConversationID: conversationID,
			Peer:           peer,
			SessionKey:     sessionKey,
			Content:        content,
		})
	}
	for _, ability := range shared.SortedStringMapKeys(questionsByAbility) {
		questions := questionsByAbility[ability]
		for i, question := range questions {
			query := stringutil.FirstNonEmpty(question.Question, question.Query, question.Prompt)
			if !shared.HasNonEmptyTrimmed(query) {
				continue
			}
			qid := stringutil.FirstNonEmpty(question.ID, question.QID)
			if qid == "" {
				qid = fmt.Sprintf("%s-%s-%03d", idPrefix, strings.ToLower(ability), i+1)
			}
			relevantIDs, err := beamQuestionRelevantIDs(question, messageIDs)
			if err != nil {
				return nil, fmt.Errorf("goncho-bench: convert BEAM question %q: %w", qid, err)
			}
			expectedNoAnswer := question.ExpectedNoAnswer || (strings.EqualFold(ability, "ABS") && len(relevantIDs) == 0)
			out = append(out, beamJSONLRecord{
				Type:                  "question",
				ID:                    qid,
				ConversationID:        conversationID,
				Scale:                 scale,
				Peer:                  peer,
				SessionKey:            sessionKey,
				Ability:               shared.NormalizeAbility(ability),
				Query:                 query,
				IdealAnswer:           stringutil.FirstNonEmpty(question.IdealAnswer, question.IdealResponse, question.Answer, question.IdealSummary),
				Rubric:                append([]string(nil), question.Rubric...),
				RelevantIDs:           relevantIDs,
				RequiredEvidenceKinds: append([]string(nil), question.RequiredEvidenceKinds...),
				ExpectedNoAnswer:      expectedNoAnswer,
				Limit:                 question.Limit,
				MaxTokens:             question.MaxTokens,
			})
		}
	}
	return out, nil
}

func beamHuggingFaceMessages(record beamHuggingFaceRecord) ([]beamConvertedMessage, error) {
	if !shared.JSONRawIsEmptyOrNull(record.Chat) {
		return flattenBeamChat(record.Chat)
	}
	return flattenBeamPlans(record.Plans)
}

func flattenBeamPlans(raw json.RawMessage) ([]beamConvertedMessage, error) {
	if shared.JSONRawIsEmptyOrNull(raw) {
		return nil, nil
	}
	var plans []struct {
		Chat json.RawMessage `json:"chat"`
	}
	if err := json.Unmarshal(raw, &plans); err != nil {
		return nil, err
	}
	out := []beamConvertedMessage{}
	for _, plan := range plans {
		messages, err := flattenBeamChat(plan.Chat)
		if err != nil {
			return nil, err
		}
		out = append(out, messages...)
	}
	return out, nil
}

func flattenBeamChat(raw json.RawMessage) ([]beamConvertedMessage, error) {
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, err
	}
	out := []beamConvertedMessage{}
	for _, item := range items {
		trimmed := shared.TrimJSONRaw(item)
		if shared.JSONRawIsEmptyOrNull(trimmed) {
			continue
		}
		switch trimmed[0] {
		case '[':
			messages, err := flattenBeamChat(item)
			if err != nil {
				return nil, err
			}
			out = append(out, messages...)
		case '{':
			var msg struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			}
			if err := json.Unmarshal(item, &msg); err != nil {
				return nil, err
			}
			if shared.HasNonEmptyTrimmed(msg.Content) {
				out = append(out, beamConvertedMessage{Role: msg.Role, Content: msg.Content})
			}
		}
	}
	return out, nil
}

func parseBeamHuggingFaceQuestions(raw json.RawMessage) (map[string][]beamConvertedQuestion, error) {
	trimmed := shared.TrimJSONRaw(raw)
	if shared.JSONRawIsEmptyOrNull(trimmed) {
		return map[string][]beamConvertedQuestion{}, nil
	}
	var parsed map[string][]beamConvertedQuestion
	if trimmed[0] == '"' {
		var encoded string
		if err := json.Unmarshal(trimmed, &encoded); err != nil {
			return nil, err
		}
		if !shared.HasNonEmptyTrimmed(encoded) {
			return map[string][]beamConvertedQuestion{}, nil
		}
		candidate := []byte(encoded)
		if !json.Valid(candidate) {
			candidate = []byte(pythonLiteralToJSONish(encoded))
		}
		if err := json.Unmarshal(candidate, &parsed); err != nil {
			return nil, err
		}
		return normalizeBeamQuestionAbilityMap(parsed), nil
	}
	if err := json.Unmarshal(trimmed, &parsed); err != nil {
		return nil, err
	}
	return normalizeBeamQuestionAbilityMap(parsed), nil
}

func normalizeBeamQuestionAbilityMap(in map[string][]beamConvertedQuestion) map[string][]beamConvertedQuestion {
	out := map[string][]beamConvertedQuestion{}
	for ability, questions := range in {
		ability = shared.NormalizeAbility(ability)
		if ability != "" {
			out[ability] = questions
		}
	}
	return out
}

func beamQuestionRelevantIDs(question beamConvertedQuestion, memoryIDs []string) ([]string, error) {
	if len(question.RelevantIDs) > 0 {
		return append([]string(nil), question.RelevantIDs...), nil
	}
	indices := question.RelevantMessageIdxs
	if len(indices) == 0 {
		indices = question.EvidenceMessageIdxs
	}
	if len(indices) == 0 {
		indices = question.SourceMessageIdxs
	}
	out := make([]string, 0, len(indices))
	seen := map[string]struct{}{}
	for _, idx := range indices {
		if idx < 0 || idx >= len(memoryIDs) {
			return nil, fmt.Errorf("message index %d out of range 0..%d", idx, len(memoryIDs)-1)
		}
		id := memoryIDs[idx]
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}

func writeConvertedBeamJSONL(path string, records []beamJSONLRecord) error {
	if path == "-" {
		return encodeBeamJSONL(os.Stdout, records)
	}
	return shared.WriteJSONLFileWithParents(path, "goncho-bench: create converted BEAM JSONL dir", "goncho-bench: create converted BEAM JSONL", "goncho-bench: write converted BEAM JSONL", records)
}

func encodeBeamJSONLBytes(records []beamJSONLRecord) ([]byte, error) {
	var buf bytes.Buffer
	if err := encodeBeamJSONL(&buf, records); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func encodeBeamJSONL(w io.Writer, records []beamJSONLRecord) error {
	return shared.WriteJSONLRows(w, records, "goncho-bench: write converted BEAM JSONL")
}

func summarizeBeamConversionRecords(records []beamJSONLRecord) beamConversionDiagnostics {
	return convertpolicy.SummarizeRecords(records)
}

func beamQuestionCount(questionsByAbility map[string][]beamConvertedQuestion) int {
	total := 0
	for _, questions := range questionsByAbility {
		total += len(questions)
	}
	return total
}

func stableBeamIDSegment(value string) string {
	return convertpolicy.StableIDSegment(value)
}

func pythonLiteralToJSONish(input string) string {
	return convertpolicy.PythonLiteralToJSONish(input)
}

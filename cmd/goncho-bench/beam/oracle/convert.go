package oracle

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

const beamConvertDefaultPeer = "beam"

type beamHuggingFaceRecord struct {
	ConversationID string          `json:"conversation_id"`
	Scale          string          `json:"scale"`
	Chat           json.RawMessage `json:"chat"`
	Plans          json.RawMessage `json:"plans"`
	Questions      json.RawMessage `json:"probing_questions"`
}

type beamConvertedMessage struct {
	Role    string
	Content string
}

type beamConvertedQuestion struct {
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

type beamConversionDiagnostics struct {
	Source                        string                     `json:"source"`
	SourceSHA256                  string                     `json:"source_sha256,omitempty"`
	ConvertedJSONLSHA256          string                     `json:"converted_jsonl_sha256,omitempty"`
	ConversationCount             int                        `json:"conversation_count"`
	MemoryCount                   int                        `json:"memory_count"`
	QuestionCount                 int                        `json:"question_count"`
	ExpectedNoAnswerQuestionCount int                        `json:"expected_no_answer_question_count"`
	UnscorableQuestionCount       int                        `json:"unscorable_question_count"`
	QuestionsByAbility            map[string]int             `json:"questions_by_ability,omitempty"`
	UnscorableByAbility           map[string]int             `json:"unscorable_by_ability,omitempty"`
	Warnings                      []beamConversionDiagnostic `json:"warnings,omitempty"`
}

type beamConversionDiagnostic struct {
	Code           string `json:"code"`
	ConversationID string `json:"conversation_id,omitempty"`
	QID            string `json:"qid,omitempty"`
	Ability        string `json:"ability,omitempty"`
	Message        string `json:"message,omitempty"`
}

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

	fallbackScale = strings.TrimSpace(fallbackScale)
	if fallbackScale == "" {
		fallbackScale = beamServiceScale
	}
	out := []beamJSONLRecord{{Type: "meta", Dataset: "beam-huggingface-converted", Scale: fallbackScale}}
	scanner := bufio.NewScanner(io.TeeReader(file, sourceHasher))
	scanner.Buffer(make([]byte, 0, 1024*1024), 16*1024*1024)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record beamHuggingFaceRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			return nil, beamConversionDiagnostics{}, fmt.Errorf("goncho-bench: decode HuggingFace BEAM line %d: %w", lineNo, err)
		}
		converted, err := convertBeamHuggingFaceRecord(record, lineNo, fallbackScale)
		if err != nil {
			return nil, beamConversionDiagnostics{}, err
		}
		out = append(out, converted...)
	}
	if err := scanner.Err(); err != nil {
		return nil, beamConversionDiagnostics{}, fmt.Errorf("goncho-bench: read HuggingFace BEAM JSONL: %w", err)
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
	diagnostics.ConvertedJSONLSHA256 = checksumBytesSHA256(convertedRaw)
	return out, diagnostics, nil
}

func convertBeamHuggingFaceRecord(record beamHuggingFaceRecord, lineNo int, fallbackScale string) ([]beamJSONLRecord, error) {
	conversationID := strings.TrimSpace(record.ConversationID)
	if conversationID == "" {
		conversationID = fmt.Sprintf("beam-conversation-%06d", lineNo)
	}
	scale := strings.TrimSpace(record.Scale)
	if scale == "" {
		scale = fallbackScale
	}
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
	abilities := make([]string, 0, len(questionsByAbility))
	for ability := range questionsByAbility {
		abilities = append(abilities, ability)
	}
	sort.Strings(abilities)
	for _, ability := range abilities {
		questions := questionsByAbility[ability]
		for i, question := range questions {
			query := firstNonEmpty(question.Question, question.Query, question.Prompt)
			if strings.TrimSpace(query) == "" {
				continue
			}
			qid := firstNonEmpty(question.ID, question.QID)
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
				Ability:               strings.ToUpper(strings.TrimSpace(ability)),
				Query:                 query,
				IdealAnswer:           firstNonEmpty(question.IdealAnswer, question.IdealResponse, question.Answer, question.IdealSummary),
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
	if len(bytes.TrimSpace(record.Chat)) > 0 && string(bytes.TrimSpace(record.Chat)) != "null" {
		return flattenBeamChat(record.Chat)
	}
	return flattenBeamPlans(record.Plans)
}

func flattenBeamPlans(raw json.RawMessage) ([]beamConvertedMessage, error) {
	if len(bytes.TrimSpace(raw)) == 0 || string(bytes.TrimSpace(raw)) == "null" {
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
		trimmed := bytes.TrimSpace(item)
		if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
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
			if strings.TrimSpace(msg.Content) != "" {
				out = append(out, beamConvertedMessage{Role: msg.Role, Content: msg.Content})
			}
		}
	}
	return out, nil
}

func parseBeamHuggingFaceQuestions(raw json.RawMessage) (map[string][]beamConvertedQuestion, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return map[string][]beamConvertedQuestion{}, nil
	}
	var parsed map[string][]beamConvertedQuestion
	if trimmed[0] == '"' {
		var encoded string
		if err := json.Unmarshal(trimmed, &encoded); err != nil {
			return nil, err
		}
		if strings.TrimSpace(encoded) == "" {
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
		ability = strings.ToUpper(strings.TrimSpace(ability))
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
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("goncho-bench: create converted BEAM JSONL dir: %w", err)
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("goncho-bench: create converted BEAM JSONL: %w", err)
	}
	defer file.Close()
	return encodeBeamJSONL(file, records)
}

func encodeBeamJSONLBytes(records []beamJSONLRecord) ([]byte, error) {
	var buf bytes.Buffer
	if err := encodeBeamJSONL(&buf, records); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func encodeBeamJSONL(w io.Writer, records []beamJSONLRecord) error {
	encoder := json.NewEncoder(w)
	for _, record := range records {
		if err := encoder.Encode(record); err != nil {
			return fmt.Errorf("goncho-bench: write converted BEAM JSONL: %w", err)
		}
	}
	return nil
}

func checksumBytesSHA256(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func summarizeBeamConversionRecords(records []beamJSONLRecord) beamConversionDiagnostics {
	diagnostics := beamConversionDiagnostics{
		Source:              "huggingface-beam-jsonl",
		QuestionsByAbility:  map[string]int{},
		UnscorableByAbility: map[string]int{},
		Warnings:            []beamConversionDiagnostic{},
	}
	conversations := map[string]struct{}{}
	for _, record := range records {
		switch strings.ToLower(strings.TrimSpace(record.Type)) {
		case "memory":
			diagnostics.MemoryCount++
			conversationID := normalizeBeamJSONLConversationID(record.ConversationID)
			conversations[conversationID] = struct{}{}
		case "question":
			diagnostics.QuestionCount++
			conversationID := normalizeBeamJSONLConversationID(record.ConversationID)
			conversations[conversationID] = struct{}{}
			ability := strings.ToUpper(strings.TrimSpace(record.Ability))
			if ability == "" {
				ability = "UNKNOWN"
			}
			diagnostics.QuestionsByAbility[ability]++
			if record.ExpectedNoAnswer {
				diagnostics.ExpectedNoAnswerQuestionCount++
				continue
			}
			if len(record.RelevantIDs) == 0 && len(record.ContextContains) == 0 {
				diagnostics.UnscorableQuestionCount++
				diagnostics.UnscorableByAbility[ability]++
				diagnostics.Warnings = append(diagnostics.Warnings, beamConversionDiagnostic{
					Code:           "beam_question_missing_relevant_ids",
					ConversationID: conversationID,
					QID:            strings.TrimSpace(record.ID),
					Ability:        ability,
					Message:        "question has no stable relevant_ids/context_contains, so stable-ID pure recall scoring treats it as unscorable",
				})
			}
		}
	}
	diagnostics.ConversationCount = len(conversations)
	return diagnostics
}

func beamQuestionCount(questionsByAbility map[string][]beamConvertedQuestion) int {
	total := 0
	for _, questions := range questionsByAbility {
		total += len(questions)
	}
	return total
}

func stableBeamIDSegment(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "conversation"
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

var pythonLiteralBarewordPattern = regexp.MustCompile(`\b(True|False|None)\b`)

func pythonLiteralToJSONish(input string) string {
	var b strings.Builder
	inString := false
	var quote rune
	escaped := false
	for _, r := range input {
		if inString {
			if escaped {
				switch r {
				case '\'', '"':
					if r == '"' {
						b.WriteString(`\"`)
					} else {
						b.WriteRune(r)
					}
				case '\\':
					b.WriteString(`\\`)
				case 'n':
					b.WriteString(`\n`)
				case 't':
					b.WriteString(`\t`)
				default:
					b.WriteRune(r)
				}
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == quote {
				b.WriteByte('"')
				inString = false
				continue
			}
			if r == '"' {
				b.WriteString(`\"`)
				continue
			}
			b.WriteRune(r)
			continue
		}
		if r == '\'' || r == '"' {
			inString = true
			quote = r
			b.WriteByte('"')
			continue
		}
		b.WriteRune(r)
	}
	return pythonLiteralBarewordPattern.ReplaceAllStringFunc(b.String(), func(token string) string {
		switch token {
		case "True":
			return "true"
		case "False":
			return "false"
		case "None":
			return "null"
		default:
			return strconv.Quote(token)
		}
	})
}

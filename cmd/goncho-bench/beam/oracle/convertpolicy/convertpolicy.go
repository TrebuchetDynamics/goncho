package convertpolicy

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/jsonlcontract"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared"
)

const DefaultPeer = "beam"

type Diagnostics struct {
	Source                        string                 `json:"source"`
	SourceSHA256                  string                 `json:"source_sha256,omitempty"`
	ConvertedJSONLSHA256          string                 `json:"converted_jsonl_sha256,omitempty"`
	ConversationCount             int                    `json:"conversation_count"`
	MemoryCount                   int                    `json:"memory_count"`
	QuestionCount                 int                    `json:"question_count"`
	ExpectedNoAnswerQuestionCount int                    `json:"expected_no_answer_question_count"`
	UnscorableQuestionCount       int                    `json:"unscorable_question_count"`
	QuestionsByAbility            map[string]int         `json:"questions_by_ability,omitempty"`
	UnscorableByAbility           map[string]int         `json:"unscorable_by_ability,omitempty"`
	Warnings                      []ConversionDiagnostic `json:"warnings,omitempty"`
}

type ConversionDiagnostic struct {
	Code           string `json:"code"`
	ConversationID string `json:"conversation_id,omitempty"`
	QID            string `json:"qid,omitempty"`
	Ability        string `json:"ability,omitempty"`
	Message        string `json:"message,omitempty"`
}

func StableIDSegment(value string) string {
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
	return shared.FirstNonEmptyTrimmed(strings.Trim(b.String(), "-"), "conversation")
}

var pythonLiteralBarewordPattern = regexp.MustCompile(`\b(True|False|None)\b`)

func SummarizeRecords(records []jsonlcontract.Record) Diagnostics {
	diagnostics := Diagnostics{
		Source:              "huggingface-beam-jsonl",
		QuestionsByAbility:  map[string]int{},
		UnscorableByAbility: map[string]int{},
		Warnings:            []ConversionDiagnostic{},
	}
	conversations := map[string]struct{}{}
	for _, record := range records {
		switch shared.NormalizeRecordType(record.Type) {
		case "memory":
			diagnostics.MemoryCount++
			conversationID := jsonlcontract.NormalizeConversationID(record.ConversationID)
			conversations[conversationID] = struct{}{}
		case "question":
			diagnostics.QuestionCount++
			conversationID := jsonlcontract.NormalizeConversationID(record.ConversationID)
			conversations[conversationID] = struct{}{}
			ability := shared.NormalizeAbility(record.Ability)
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
				diagnostics.Warnings = append(diagnostics.Warnings, ConversionDiagnostic{
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

func PythonLiteralToJSONish(input string) string {
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

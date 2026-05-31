package oracle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared"
	"github.com/TrebuchetDynamics/goncho/service"
)

type beamServiceJudgment struct {
	Scale          string   `json:"scale,omitempty"`
	ConversationID string   `json:"conversation_id,omitempty"`
	QID            string   `json:"qid"`
	Ability        string   `json:"ability,omitempty"`
	Question       string   `json:"question,omitempty"`
	AIAnswer       string   `json:"ai_answer,omitempty"`
	Score          float64  `json:"score"`
	Nuggets        []string `json:"nuggets,omitempty"`
	Assessment     string   `json:"assessment,omitempty"`
	AnswerTimeMS   float64  `json:"answer_time_ms,omitempty"`
	JudgeTimeMS    float64  `json:"judge_time_ms,omitempty"`
}

type beamServiceJudgmentSet struct {
	Source       string
	SourceSHA256 string
	Rows         map[shared.OutcomeKey]beamServiceJudgment
	QuestionRows map[shared.QuestionKey]beamServiceJudgment
	RowCount     int
}

type beamServiceJudgmentDiagnostics struct {
	Source         string   `json:"source"`
	SourceSHA256   string   `json:"source_sha256,omitempty"`
	RowCount       int      `json:"row_count"`
	AppliedCount   int      `json:"applied_count"`
	MissingCount   int      `json:"missing_count"`
	UnmatchedCount int      `json:"unmatched_count"`
	MissingQIDs    []string `json:"missing_qids,omitempty"`
	UnmatchedQIDs  []string `json:"unmatched_qids,omitempty"`
}

func loadBeamServiceJudgments(path string) (*beamServiceJudgmentSet, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("goncho-bench: open BEAM service judgments: %w", err)
	}
	sourceSHA256 := shared.ChecksumBytesSHA256(raw)
	rows := map[shared.OutcomeKey]beamServiceJudgment{}
	questionRows := map[shared.QuestionKey]beamServiceJudgment{}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) > 0 && trimmed[0] == '{' && bytes.Contains(trimmed, []byte(`"results"`)) {
		if err := loadNestedBeamServiceJudgments(trimmed, rows, questionRows); err != nil {
			return nil, err
		}
	} else if err := loadJSONLBeamServiceJudgments(raw, rows, questionRows); err != nil {
		return nil, err
	}
	return &beamServiceJudgmentSet{Source: "beam-service-judgments", SourceSHA256: sourceSHA256, Rows: rows, QuestionRows: questionRows, RowCount: len(rows)}, nil
}

func loadJSONLBeamServiceJudgments(raw []byte, rows map[shared.OutcomeKey]beamServiceJudgment, questionRows map[shared.QuestionKey]beamServiceJudgment) error {
	scanner := shared.NewJSONLScanner(bytes.NewReader(raw))
	return shared.ForEachNonEmptyJSONLLine(scanner, "goncho-bench: read BEAM service judgments", func(lineNo int, line string) error {
		var row beamServiceJudgment
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return fmt.Errorf("goncho-bench: decode BEAM service judgment line %d: %w", lineNo, err)
		}
		return addBeamServiceJudgment(rows, questionRows, row, fmt.Sprintf("line %d", lineNo))
	})
}

func loadNestedBeamServiceJudgments(raw []byte, rows map[shared.OutcomeKey]beamServiceJudgment, questionRows map[shared.QuestionKey]beamServiceJudgment) error {
	var file struct {
		Results []struct {
			Scale          string                `json:"scale"`
			ConversationID string                `json:"conversation_id"`
			Results        []beamServiceJudgment `json:"results"`
		} `json:"results"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		return fmt.Errorf("goncho-bench: decode nested BEAM service judgments: %w", err)
	}
	for conversationIndex, conv := range file.Results {
		for resultIndex, row := range conv.Results {
			if !shared.HasNonEmptyTrimmed(row.Scale) {
				row.Scale = conv.Scale
			}
			if !shared.HasNonEmptyTrimmed(row.ConversationID) {
				row.ConversationID = conv.ConversationID
			}
			if err := addBeamServiceJudgment(rows, questionRows, row, fmt.Sprintf("conversation %d result %d", conversationIndex+1, resultIndex+1)); err != nil {
				return err
			}
		}
	}
	return nil
}

func addBeamServiceJudgment(rows map[shared.OutcomeKey]beamServiceJudgment, questionRows map[shared.QuestionKey]beamServiceJudgment, row beamServiceJudgment, location string) error {
	row.Scale = strings.TrimSpace(row.Scale)
	row.ConversationID = strings.TrimSpace(row.ConversationID)
	row.QID = strings.TrimSpace(row.QID)
	row.Ability = shared.NormalizeAbility(row.Ability)
	row.Question = strings.TrimSpace(row.Question)
	if row.QID == "" {
		return fmt.Errorf("goncho-bench: BEAM service judgment %s missing qid", location)
	}
	rows[shared.NewOutcomeKey(row.Scale, row.ConversationID, row.QID)] = row
	if row.Question != "" {
		questionRows[shared.NewQuestionKey(row.Scale, row.ConversationID, row.Ability, row.Question)] = row
		questionRows[shared.NewQuestionKey(row.Scale, row.ConversationID, "", row.Question)] = row
	}
	return nil
}

func (s *beamServiceJudgmentSet) find(c goncho.RecallBenchmarkCaseReport) (beamServiceJudgment, bool) {
	if s == nil {
		return beamServiceJudgment{}, false
	}
	qid := strings.TrimSpace(c.ID)
	for _, key := range []shared.OutcomeKey{
		shared.NewOutcomeKey(beamServiceCaseScale(c), beamServiceCaseConversationID(c), qid),
		shared.NewOutcomeKey("", beamServiceCaseConversationID(c), qid),
		shared.NewOutcomeKey(beamServiceCaseScale(c), "", qid),
		shared.NewOutcomeKey("", "", qid),
	} {
		if row, ok := s.Rows[key]; ok {
			return row, true
		}
	}
	ability := shared.NormalizeAbility(c.Ability)
	question := strings.TrimSpace(c.Question)
	if question == "" {
		return beamServiceJudgment{}, false
	}
	for _, key := range []shared.QuestionKey{
		shared.NewQuestionKey(beamServiceCaseScale(c), beamServiceCaseConversationID(c), ability, question),
		shared.NewQuestionKey("", beamServiceCaseConversationID(c), ability, question),
		shared.NewQuestionKey(beamServiceCaseScale(c), "", ability, question),
		shared.NewQuestionKey("", "", ability, question),
		shared.NewQuestionKey(beamServiceCaseScale(c), beamServiceCaseConversationID(c), "", question),
		shared.NewQuestionKey("", beamServiceCaseConversationID(c), "", question),
		shared.NewQuestionKey(beamServiceCaseScale(c), "", "", question),
		shared.NewQuestionKey("", "", "", question),
	} {
		if row, ok := s.QuestionRows[key]; ok {
			return row, true
		}
	}
	return beamServiceJudgment{}, false
}

func (s *beamServiceJudgmentSet) diagnostics(report goncho.RecallBenchmarkReport) beamServiceJudgmentDiagnostics {
	diag := beamServiceJudgmentDiagnostics{Source: s.Source, SourceSHA256: s.SourceSHA256, RowCount: s.RowCount}
	matched := map[shared.OutcomeKey]struct{}{}
	for _, c := range report.Cases {
		if row, ok := s.find(c); ok {
			diag.AppliedCount++
			matched[shared.NewOutcomeKey(row.Scale, row.ConversationID, row.QID)] = struct{}{}
			continue
		}
		diag.MissingCount++
		if len(diag.MissingQIDs) < 10 {
			diag.MissingQIDs = append(diag.MissingQIDs, c.ID)
		}
	}
	for key, row := range s.Rows {
		if _, ok := matched[key]; ok {
			continue
		}
		diag.UnmatchedCount++
		if len(diag.UnmatchedQIDs) < 10 {
			diag.UnmatchedQIDs = append(diag.UnmatchedQIDs, row.QID)
		}
	}
	return diag
}

func requireCompleteBeamServiceJudgments(judgments beamServiceJudgmentSet, report goncho.RecallBenchmarkReport) error {
	diag := judgments.diagnostics(report)
	if diag.MissingCount == 0 && diag.UnmatchedCount == 0 {
		return nil
	}
	return fmt.Errorf("goncho-bench: BEAM service judgments incomplete: missing=%d unmatched=%d missing_qids=%s unmatched_qids=%s", diag.MissingCount, diag.UnmatchedCount, strings.Join(diag.MissingQIDs, ","), strings.Join(diag.UnmatchedQIDs, ","))
}

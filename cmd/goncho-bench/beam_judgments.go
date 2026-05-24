package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/TrebuchetDynamics/goncho"
)

type beamServiceJudgment struct {
	Scale          string   `json:"scale,omitempty"`
	ConversationID string   `json:"conversation_id,omitempty"`
	QID            string   `json:"qid"`
	AIAnswer       string   `json:"ai_answer,omitempty"`
	Score          float64  `json:"score"`
	Nuggets        []string `json:"nuggets,omitempty"`
	Assessment     string   `json:"assessment,omitempty"`
	AnswerTimeMS   int      `json:"answer_time_ms,omitempty"`
	JudgeTimeMS    int      `json:"judge_time_ms,omitempty"`
}

type beamServiceJudgmentSet struct {
	Source       string
	SourceSHA256 string
	Rows         map[string]beamServiceJudgment
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
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("goncho-bench: open BEAM service judgments: %w", err)
	}
	defer file.Close()
	hasher := sha256.New()
	scanner := bufio.NewScanner(io.TeeReader(file, hasher))
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	rows := map[string]beamServiceJudgment{}
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var row beamServiceJudgment
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, fmt.Errorf("goncho-bench: decode BEAM service judgment line %d: %w", lineNo, err)
		}
		row.Scale = strings.TrimSpace(row.Scale)
		row.ConversationID = strings.TrimSpace(row.ConversationID)
		row.QID = strings.TrimSpace(row.QID)
		if row.QID == "" {
			return nil, fmt.Errorf("goncho-bench: BEAM service judgment line %d missing qid", lineNo)
		}
		rows[beamServiceJudgmentKey(row.Scale, row.ConversationID, row.QID)] = row
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("goncho-bench: read BEAM service judgments: %w", err)
	}
	return &beamServiceJudgmentSet{Source: "beam-service-judgments-jsonl", SourceSHA256: hex.EncodeToString(hasher.Sum(nil)), Rows: rows, RowCount: len(rows)}, nil
}

func (s *beamServiceJudgmentSet) find(c goncho.RecallBenchmarkCaseReport) (beamServiceJudgment, bool) {
	if s == nil {
		return beamServiceJudgment{}, false
	}
	qid := strings.TrimSpace(c.ID)
	for _, key := range []string{
		beamServiceJudgmentKey(beamServiceCaseScale(c), beamServiceCaseConversationID(c), qid),
		beamServiceJudgmentKey("", beamServiceCaseConversationID(c), qid),
		beamServiceJudgmentKey(beamServiceCaseScale(c), "", qid),
		beamServiceJudgmentKey("", "", qid),
	} {
		if row, ok := s.Rows[key]; ok {
			return row, true
		}
	}
	return beamServiceJudgment{}, false
}

func (s *beamServiceJudgmentSet) diagnostics(report goncho.RecallBenchmarkReport) beamServiceJudgmentDiagnostics {
	diag := beamServiceJudgmentDiagnostics{Source: s.Source, SourceSHA256: s.SourceSHA256, RowCount: s.RowCount}
	matched := map[string]struct{}{}
	for _, c := range report.Cases {
		if row, ok := s.find(c); ok {
			diag.AppliedCount++
			matched[beamServiceJudgmentKey(row.Scale, row.ConversationID, row.QID)] = struct{}{}
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

func beamServiceJudgmentKey(scale, conversationID, qid string) string {
	return strings.TrimSpace(scale) + "\x00" + strings.TrimSpace(conversationID) + "\x00" + strings.TrimSpace(qid)
}

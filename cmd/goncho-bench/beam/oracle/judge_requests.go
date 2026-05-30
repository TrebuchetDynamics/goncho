package oracle

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/service"
)

const (
	beamAnswerSystemPrompt = "You answer BEAM memory benchmark questions using only the retrieved memory context. If the context is insufficient, say you do not know."
	beamJudgeSystemPrompt  = "You are an expert evaluator for a memory benchmark. Score the AI answer against each rubric item and return JSON with scores and overall_score."
	beamAnswerPlaceholder  = "[AI_ANSWER]"
)

type beamServiceJudgeRequestRow struct {
	ConfigID             string                      `json:"config_id"`
	RunStartedAt         string                      `json:"run_started_at"`
	Scale                string                      `json:"scale"`
	ConversationID       string                      `json:"conversation_id"`
	QID                  string                      `json:"qid"`
	Ability              string                      `json:"ability"`
	Question             string                      `json:"question"`
	PureRecall           bool                        `json:"pure_recall"`
	AnswerRequest        beamServicePromptRequest    `json:"answer_request"`
	JudgeRequest         beamServiceJudgePrompt      `json:"judge_request"`
	RecallProvenance     beamServiceRecallProvenance `json:"recall_provenance"`
	CandidateMemoryIDs   []string                    `json:"candidate_memory_ids,omitempty"`
	SelectedMemoryIDs    []string                    `json:"selected_memory_ids,omitempty"`
	RubricContextScore   float64                     `json:"rubric_context_score,omitempty"`
	RubricContextMatches []string                    `json:"rubric_context_matches,omitempty"`
}

type beamServicePromptRequest struct {
	System  string `json:"system"`
	User    string `json:"user"`
	Context string `json:"context"`
}

type beamServiceJudgePrompt struct {
	System            string   `json:"system"`
	User              string   `json:"user"`
	Question          string   `json:"question"`
	IdealAnswer       string   `json:"ideal_answer,omitempty"`
	Rubric            []string `json:"rubric,omitempty"`
	AnswerPlaceholder string   `json:"answer_placeholder"`
}

func writeBeamServiceJudgeRequests(path string, report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("goncho-bench: create BEAM judge request dir: %w", err)
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("goncho-bench: create BEAM judge requests: %w", err)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	for _, row := range buildBeamServiceJudgeRequestRows(report, configID, runStartedAt) {
		if err := encoder.Encode(row); err != nil {
			return fmt.Errorf("goncho-bench: write BEAM judge request row: %w", err)
		}
	}
	return nil
}

func buildBeamServiceJudgeRequestRows(report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time) []beamServiceJudgeRequestRow {
	out := make([]beamServiceJudgeRequestRow, 0, len(report.Cases))
	started := runStartedAt.UTC().Format(beamServicePairedDateTimeFormat)
	for _, c := range report.Cases {
		question := strings.TrimSpace(c.Question)
		context := strings.TrimSpace(c.SelectedContext)
		out = append(out, beamServiceJudgeRequestRow{
			ConfigID:             configID,
			RunStartedAt:         started,
			Scale:                beamServiceCaseScale(c),
			ConversationID:       beamServiceCaseConversationID(c),
			QID:                  c.ID,
			Ability:              strings.ToUpper(strings.TrimSpace(c.Ability)),
			Question:             question,
			PureRecall:           true,
			AnswerRequest:        buildBeamServiceAnswerRequest(question, context),
			JudgeRequest:         buildBeamServiceJudgePrompt(question, c.IdealAnswer, c.Rubric),
			RecallProvenance:     beamServiceCaseRecallProvenance(c),
			CandidateMemoryIDs:   append([]string(nil), c.CandidateMemoryIDs...),
			SelectedMemoryIDs:    append([]string(nil), c.SelectedMemoryIDs...),
			RubricContextScore:   c.RubricContextScore,
			RubricContextMatches: append([]string(nil), c.RubricContextMatches...),
		})
	}
	return out
}

func buildBeamServiceAnswerRequest(question, context string) beamServicePromptRequest {
	if strings.TrimSpace(context) == "" {
		context = "[No memories found]"
	}
	return beamServicePromptRequest{
		System:  beamAnswerSystemPrompt,
		User:    fmt.Sprintf("RETRIEVED MEMORIES:\n%s\n\nQUESTION: %s\n\nANSWER:", strings.TrimSpace(context), strings.TrimSpace(question)),
		Context: strings.TrimSpace(context),
	}
}

func buildBeamServiceJudgePrompt(question, idealAnswer string, rubric []string) beamServiceJudgePrompt {
	rubricText := ""
	if len(rubric) > 0 {
		var b strings.Builder
		for i, item := range rubric {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			fmt.Fprintf(&b, "%d. %s", i+1, item)
		}
		rubricText = b.String()
	}
	user := fmt.Sprintf("QUESTION: %s\n\nRUBRIC ITEMS:\n%s\n\nAI's ANSWER: %s\n\nFor each rubric item, score how well the AI's answer matches. Return JSON with scores array and overall_score.", strings.TrimSpace(question), rubricText, beamAnswerPlaceholder)
	return beamServiceJudgePrompt{
		System:            beamJudgeSystemPrompt,
		User:              user,
		Question:          strings.TrimSpace(question),
		IdealAnswer:       strings.TrimSpace(idealAnswer),
		Rubric:            append([]string(nil), rubric...),
		AnswerPlaceholder: beamAnswerPlaceholder,
	}
}

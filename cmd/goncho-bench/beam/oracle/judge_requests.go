package oracle

import (
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/artifactcontract"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/judgeprompt"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared"
	"github.com/TrebuchetDynamics/goncho/service"
)

const (
	beamAnswerSystemPrompt = judgeprompt.AnswerSystemPrompt
	beamJudgeSystemPrompt  = judgeprompt.JudgeSystemPrompt
	beamAnswerPlaceholder  = judgeprompt.AnswerPlaceholder
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

type beamServicePromptRequest = judgeprompt.AnswerRequest

type beamServiceJudgePrompt = judgeprompt.JudgePrompt

func writeBeamServiceJudgeRequests(path string, report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time) error {
	return shared.WriteJSONLFileWithParents(path, "goncho-bench: create BEAM judge request dir", "goncho-bench: create BEAM judge requests", "goncho-bench: write BEAM judge request row", buildBeamServiceJudgeRequestRows(report, configID, runStartedAt))
}

func buildBeamServiceJudgeRequestRows(report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time) []beamServiceJudgeRequestRow {
	out := make([]beamServiceJudgeRequestRow, 0, len(report.Cases))
	started := shared.FormatArtifactTimestamp(runStartedAt)
	for _, c := range report.Cases {
		fields := artifactcontract.BuildCaseFields(c)
		context := strings.TrimSpace(c.SelectedContext)
		out = append(out, beamServiceJudgeRequestRow{
			ConfigID:             configID,
			RunStartedAt:         started,
			Scale:                fields.Scale,
			ConversationID:       fields.ConversationID,
			QID:                  fields.QID,
			Ability:              fields.Ability,
			Question:             fields.Question,
			PureRecall:           true,
			AnswerRequest:        buildBeamServiceAnswerRequest(fields.Question, context),
			JudgeRequest:         buildBeamServiceJudgePrompt(fields.Question, c.IdealAnswer, c.Rubric),
			RecallProvenance:     beamServiceCaseRecallProvenance(c),
			CandidateMemoryIDs:   fields.CandidateMemoryIDs,
			SelectedMemoryIDs:    fields.SelectedMemoryIDs,
			RubricContextScore:   fields.RubricContextScore,
			RubricContextMatches: fields.RubricContextMatches,
		})
	}
	return out
}

func buildBeamServiceAnswerRequest(question, context string) beamServicePromptRequest {
	return judgeprompt.BuildAnswerRequest(question, context)
}

func buildBeamServiceJudgePrompt(question, idealAnswer string, rubric []string) beamServiceJudgePrompt {
	return judgeprompt.BuildJudgePrompt(question, idealAnswer, rubric)
}

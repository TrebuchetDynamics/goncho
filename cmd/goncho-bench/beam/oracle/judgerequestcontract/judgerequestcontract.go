package judgerequestcontract

import (
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/artifactcontract"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/judgeprompt"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared"
	goncho "github.com/TrebuchetDynamics/goncho/service"
)

// Row is the stable JSONL contract for offline BEAM answer and judge requests.
type Row struct {
	ConfigID             string                            `json:"config_id"`
	RunStartedAt         string                            `json:"run_started_at"`
	Scale                string                            `json:"scale"`
	ConversationID       string                            `json:"conversation_id"`
	QID                  string                            `json:"qid"`
	Ability              string                            `json:"ability"`
	Question             string                            `json:"question"`
	PureRecall           bool                              `json:"pure_recall"`
	AnswerRequest        judgeprompt.AnswerRequest         `json:"answer_request"`
	JudgeRequest         judgeprompt.JudgePrompt           `json:"judge_request"`
	RecallProvenance     artifactcontract.RecallProvenance `json:"recall_provenance"`
	CandidateMemoryIDs   []string                          `json:"candidate_memory_ids,omitempty"`
	SelectedMemoryIDs    []string                          `json:"selected_memory_ids,omitempty"`
	RubricContextScore   float64                           `json:"rubric_context_score,omitempty"`
	RubricContextMatches []string                          `json:"rubric_context_matches,omitempty"`
}

// BuildRows projects a recall report into deterministic offline judge request rows.
func BuildRows(report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time) []Row {
	out := make([]Row, 0, len(report.Cases))
	started := shared.FormatArtifactTimestamp(runStartedAt)
	for _, c := range report.Cases {
		fields := artifactcontract.BuildCaseFields(c)
		context := strings.TrimSpace(c.SelectedContext)
		out = append(out, Row{
			ConfigID:             configID,
			RunStartedAt:         started,
			Scale:                fields.Scale,
			ConversationID:       fields.ConversationID,
			QID:                  fields.QID,
			Ability:              fields.Ability,
			Question:             fields.Question,
			PureRecall:           true,
			AnswerRequest:        judgeprompt.BuildAnswerRequest(fields.Question, context),
			JudgeRequest:         judgeprompt.BuildJudgePrompt(fields.Question, c.IdealAnswer, c.Rubric),
			RecallProvenance:     artifactcontract.BuildRecallProvenance(c),
			CandidateMemoryIDs:   fields.CandidateMemoryIDs,
			SelectedMemoryIDs:    fields.SelectedMemoryIDs,
			RubricContextScore:   fields.RubricContextScore,
			RubricContextMatches: fields.RubricContextMatches,
		})
	}
	return out
}

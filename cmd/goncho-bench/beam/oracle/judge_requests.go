package oracle

import (
	"time"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/judgeprompt"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/judgerequestcontract"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared"
	goncho "github.com/TrebuchetDynamics/goncho/service"
)

const (
	beamAnswerSystemPrompt = judgeprompt.AnswerSystemPrompt
	beamJudgeSystemPrompt  = judgeprompt.JudgeSystemPrompt
	beamAnswerPlaceholder  = judgeprompt.AnswerPlaceholder
)

type beamServiceJudgeRequestRow = judgerequestcontract.Row

type beamServicePromptRequest = judgeprompt.AnswerRequest

type beamServiceJudgePrompt = judgeprompt.JudgePrompt

func writeBeamServiceJudgeRequests(path string, report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time) error {
	return shared.WriteJSONLFileWithParents(path, "goncho-bench: create BEAM judge request dir", "goncho-bench: create BEAM judge requests", "goncho-bench: write BEAM judge request row", buildBeamServiceJudgeRequestRows(report, configID, runStartedAt))
}

func buildBeamServiceJudgeRequestRows(report goncho.RecallBenchmarkReport, configID string, runStartedAt time.Time) []beamServiceJudgeRequestRow {
	return judgerequestcontract.BuildRows(report, configID, runStartedAt)
}

func buildBeamServiceAnswerRequest(question, context string) beamServicePromptRequest {
	return judgeprompt.BuildAnswerRequest(question, context)
}

func buildBeamServiceJudgePrompt(question, idealAnswer string, rubric []string) beamServiceJudgePrompt {
	return judgeprompt.BuildJudgePrompt(question, idealAnswer, rubric)
}

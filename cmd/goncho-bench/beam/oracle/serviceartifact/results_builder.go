package serviceartifact

import (
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/artifactcontract"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/casecontract"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared"
	goncho "github.com/TrebuchetDynamics/goncho/service"
)

// CaseJudgment contains optional external answer/judge fields for one result row.
type CaseJudgment struct {
	Has          bool
	Score        float64
	AIAnswer     string
	Nuggets      []string
	Assessment   string
	AnswerTimeMS float64
	JudgeTimeMS  float64
}

// CaseJudgmentFunc supplies optional external judgment fields for a report case.
type CaseJudgmentFunc func(goncho.RecallBenchmarkCaseReport) CaseJudgment

// ResultsOptions contains metadata and callbacks needed to build service results.
type ResultsOptions struct {
	ConfigID    string
	RunStarted  time.Time
	JudgeModel  string
	PureRecall  bool
	Diagnostics map[string]interface{}
	Judgment    CaseJudgmentFunc
}

// BuildResults projects a recall report into the BEAM service results artifact contract.
func BuildResults(report goncho.RecallBenchmarkReport, opts ResultsOptions) ResultsFile {
	type conversationAccumulator struct {
		conversationID string
		scale          string
		results        []QuestionResult
	}
	byConversation := map[string]*conversationAccumulator{}
	conversationOrder := []string{}
	scales := map[string]struct{}{}
	for _, c := range report.Cases {
		fields := artifactcontract.BuildCaseFields(c)
		key := fields.Scale + "\x00" + fields.ConversationID
		acc := byConversation[key]
		if acc == nil {
			acc = &conversationAccumulator{conversationID: fields.ConversationID, scale: fields.Scale}
			byConversation[key] = acc
			conversationOrder = append(conversationOrder, key)
		}
		scales[fields.Scale] = struct{}{}
		judgment := CaseJudgment{}
		if opts.Judgment != nil {
			judgment = opts.Judgment(c)
		}
		score := casecontract.Score(c)
		aiAnswer := ""
		nuggets := []string{}
		assessment := casecontract.Assessment(c, score)
		answerTimeMS := 0.0
		judgeTimeMS := 0.0
		if judgment.Has {
			score = shared.RoundMetric(judgment.Score)
			aiAnswer = strings.TrimSpace(judgment.AIAnswer)
			nuggets = append([]string(nil), judgment.Nuggets...)
			assessment = strings.TrimSpace(judgment.Assessment)
			answerTimeMS = judgment.AnswerTimeMS
			judgeTimeMS = judgment.JudgeTimeMS
		}
		acc.results = append(acc.results, QuestionResult{
			QID:                  fields.QID,
			Ability:              fields.Ability,
			Question:             fields.Question,
			IdealAnswer:          strings.TrimSpace(c.IdealAnswer),
			Rubric:               append([]string(nil), c.Rubric...),
			RubricContextScore:   c.RubricContextScore,
			RubricContextMatches: fields.RubricContextMatches,
			AIAnswer:             aiAnswer,
			RecallProvenance:     artifactcontract.BuildRecallProvenance(c),
			Score:                score,
			Nuggets:              nuggets,
			Assessment:           assessment,
			AnswerTimeMS:         answerTimeMS,
			JudgeTimeMS:          judgeTimeMS,
		})
	}
	conversationResults := make([]ConversationResults, 0, len(conversationOrder))
	for _, key := range conversationOrder {
		acc := byConversation[key]
		conversationResults = append(conversationResults, ConversationResults{
			ConversationID: acc.conversationID,
			Scale:          acc.scale,
			NumQuestions:   len(acc.results),
			NumEvaluated:   len(acc.results),
			Results:        acc.results,
		})
	}
	return ResultsFile{
		Metadata: ResultsMetadata{
			Date:               time.Now().UTC().Format(time.RFC3339),
			RunStartedAt:       shared.FormatArtifactTimestamp(opts.RunStarted),
			ConfigID:           opts.ConfigID,
			Model:              casecontract.ModelName,
			JudgeModel:         opts.JudgeModel,
			TopK:               5,
			SampleSize:         len(conversationResults),
			Scales:             shared.SortedStringMapKeys(scales),
			TotalConversations: len(conversationResults),
			PureRecall:         opts.PureRecall,
			Config: map[string]any{
				"pure_recall":           opts.PureRecall,
				"external_judgments":    !opts.PureRecall,
				"allow_harness_oracles": false,
				"full_context":          false,
				"use_cloud":             false,
			},
			Diagnostics: opts.Diagnostics,
		},
		Results: conversationResults,
	}
}

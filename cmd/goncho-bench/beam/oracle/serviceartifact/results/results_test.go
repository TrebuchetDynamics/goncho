package results

import (
	"testing"
	"time"

	goncho "github.com/TrebuchetDynamics/goncho/service"
)

func TestBuildResultsGroupsConversationsAndAppliesJudgmentContract(t *testing.T) {
	started := time.Date(2026, 5, 30, 1, 2, 3, 0, time.UTC)
	report := goncho.RecallBenchmarkReport{Cases: []goncho.RecallBenchmarkCaseReport{{
		ID:                    "q1",
		Scale:                 "10K",
		ConversationID:        "conv",
		Ability:               "rw",
		Question:              "  Who? ",
		IdealAnswer:           " ideal ",
		Rubric:                []string{"rubric"},
		CandidateMemoryIDs:    []string{"mem-1"},
		SelectedEvidenceKinds: []string{"fact"},
		TopEvidenceKinds:      []string{"graph"},
		RecallAt5:             1,
		ContextSatisfied:      true,
		TokenBudgetWithin:     true,
	}}}

	results := Build(report, Options{
		ConfigID:    "cfg",
		RunStarted:  started,
		JudgeModel:  "judge",
		PureRecall:  false,
		Diagnostics: map[string]interface{}{"ok": true},
		Judgment: func(goncho.RecallBenchmarkCaseReport) CaseJudgment {
			return CaseJudgment{Has: true, Score: 0.77777, AIAnswer: " answer ", Nuggets: []string{"nug"}, Assessment: " assessed ", AnswerTimeMS: 1, JudgeTimeMS: 2}
		},
	})

	if results.Metadata.ConfigID != "cfg" || results.Metadata.RunStartedAt != "2026-05-30T01:02:03Z" || results.Metadata.JudgeModel != "judge" || results.Metadata.PureRecall || results.Metadata.SampleSize != 1 {
		t.Fatalf("metadata = %#v, want supplied result metadata", results.Metadata)
	}
	if len(results.Results) != 1 || len(results.Results[0].Results) != 1 {
		t.Fatalf("results = %#v, want one conversation with one question", results.Results)
	}
	question := results.Results[0].Results[0]
	if question.QID != "q1" || question.Ability != "RW" || question.Question != "Who?" || question.IdealAnswer != "ideal" {
		t.Fatalf("question fields = %#v, want canonical fields", question)
	}
	if question.Score != 0.7778 || question.AIAnswer != "answer" || question.Assessment != "assessed" || len(question.Nuggets) != 1 || question.Nuggets[0] != "nug" {
		t.Fatalf("judgment fields = %#v, want rounded external judgment fields", question)
	}
}

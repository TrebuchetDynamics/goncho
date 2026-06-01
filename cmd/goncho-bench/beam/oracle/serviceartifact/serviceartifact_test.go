package serviceartifact

import (
	"testing"
	"time"

	goncho "github.com/TrebuchetDynamics/goncho/service"
)

func TestFacadeKeepsPublicBuilderContracts(t *testing.T) {
	started := time.Date(2026, 5, 30, 1, 2, 3, 0, time.UTC)
	report := goncho.RecallBenchmarkReport{
		CaseCount: 1,
		Cases: []goncho.RecallBenchmarkCaseReport{{
			ID:                "q1",
			Scale:             "10K",
			ConversationID:    "conv",
			Ability:           "rw",
			Question:          " Who? ",
			RecallAt5:         1,
			ContextSatisfied:  true,
			TokenBudgetWithin: true,
		}},
	}
	score := func(goncho.RecallBenchmarkCaseReport) float64 { return 1 }

	summary := BuildSummary(report, SummaryOptions{ConfigID: "cfg", RunStarted: started, Score: score})
	if summary.Metadata.ConfigID != "cfg" || summary.AbilitySummary["10K"]["RW"].Count != 1 {
		t.Fatalf("summary facade = %#v, want delegated summary contract", summary)
	}

	results := BuildResults(report, ResultsOptions{ConfigID: "cfg", RunStarted: started, PureRecall: true})
	if results.Metadata.RunStartedAt != "2026-05-30T01:02:03Z" || len(results.Results) != 1 {
		t.Fatalf("results facade = %#v, want delegated results contract", results)
	}

	paired := BuildPairedOutcomes(report, "cfg", started, score)
	if len(paired) != 1 || !paired[0].Correct {
		t.Fatalf("paired facade = %#v, want delegated paired outcome contract", paired)
	}

	failures := BuildFailureAuditRows(report, "cfg", started)
	if len(failures) != 0 {
		t.Fatalf("failures facade = %#v, want no passing cases", failures)
	}
}

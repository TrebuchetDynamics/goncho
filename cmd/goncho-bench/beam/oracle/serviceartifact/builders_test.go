package serviceartifact

import (
	"testing"
	"time"

	goncho "github.com/TrebuchetDynamics/goncho/service"
)

func TestBuildPairedOutcomesUsesSharedCaseFieldsAndScoreContract(t *testing.T) {
	started := time.Date(2026, 5, 30, 1, 2, 3, 0, time.UTC)
	report := goncho.RecallBenchmarkReport{Cases: []goncho.RecallBenchmarkCaseReport{{
		ID:             " q-1 ",
		Scale:          " 10K ",
		ConversationID: " conv-1 ",
		Ability:        " rw ",
		Question:       "  Who?  ",
	}}}

	rows := BuildPairedOutcomes(report, "cfg", started, func(goncho.RecallBenchmarkCaseReport) float64 { return 1 })

	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	row := rows[0]
	if row.ConfigID != "cfg" || row.RunStartedAt != "2026-05-30T01:02:03Z" || row.Scale != "10K" || row.ConversationID != "conv-1" || row.QID != " q-1 " || row.Ability != "RW" || row.Question != "Who?" {
		t.Fatalf("row = %#v, want canonical paired outcome fields", row)
	}
	if row.Score != 1 || !row.Correct {
		t.Fatalf("score/correct = %v/%v, want 1/true", row.Score, row.Correct)
	}
}

func TestBuildFailureAuditRowsKeepsOnlyFailuresAndCopiesSlices(t *testing.T) {
	started := time.Date(2026, 5, 30, 1, 2, 3, 0, time.UTC)
	candidateIDs := []string{"noise", "rel-1", "other"}
	relevantIDs := []string{"rel-1"}
	report := goncho.RecallBenchmarkReport{Cases: []goncho.RecallBenchmarkCaseReport{
		{ID: "pass", RecallAt5: 1, ContextSatisfied: true, TokenBudgetWithin: true},
		{
			ID:                    "fail",
			Scale:                 "1M",
			ConversationID:        "conv-2",
			Ability:               "rsg",
			Question:              "  Why? ",
			RelevantIDs:           relevantIDs,
			CandidateMemoryIDs:    candidateIDs,
			SelectedMemoryIDs:     []string{"sel-1"},
			RequiredEvidenceKinds: []string{"graph"},
			RecallAt5:             0,
			RecallAt10:            1,
			ContextSatisfied:      true,
			ProvenanceSatisfied:   true,
			TokenBudgetWithin:     true,
			WarningCodes:          []string{"warn"},
		},
	}}

	rows := BuildFailureAuditRows(report, "cfg", started)

	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	row := rows[0]
	if row.ConfigID != "cfg" || row.RunStartedAt != "2026-05-30T01:02:03Z" || row.Scale != "1M" || row.ConversationID != "conv-2" || row.QID != "fail" || row.Ability != "RSG" || row.Question != "Why?" {
		t.Fatalf("row = %#v, want canonical failure audit fields", row)
	}
	if row.Score != 0 || row.FailureMode != "rank_too_low" || row.Rank != 2 {
		t.Fatalf("failure score/mode/rank = %v/%q/%d, want 0/rank_too_low/2", row.Score, row.FailureMode, row.Rank)
	}
	candidateIDs[1] = "mutated"
	relevantIDs[0] = "mutated"
	if row.CandidateMemoryIDs[1] != "rel-1" || row.RetrievedTop10[1] != "rel-1" || row.RelevantIDs[0] != "rel-1" {
		t.Fatalf("row slices changed after source mutation: %#v", row)
	}
}

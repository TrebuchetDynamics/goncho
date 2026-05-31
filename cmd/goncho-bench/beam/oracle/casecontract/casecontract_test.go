package casecontract

import (
	"testing"

	goncho "github.com/TrebuchetDynamics/goncho/service"
)

func TestScorePreservesRecallGateSemantics(t *testing.T) {
	base := goncho.RecallBenchmarkCaseReport{
		RecallAt5:             0.87654,
		ContextSatisfied:      true,
		TokenBudgetWithin:     true,
		ProvenanceSatisfied:   true,
		RequiredEvidenceKinds: []string{"graph"},
	}
	if got, want := Score(base), 0.8765; got != want {
		t.Fatalf("Score() = %v, want rounded recall %v", got, want)
	}

	missingContext := base
	missingContext.ContextSatisfied = false
	if got := Score(missingContext); got != 0 {
		t.Fatalf("Score() with unsatisfied context = %v, want 0", got)
	}

	missingProvenance := base
	missingProvenance.ProvenanceSatisfied = false
	if got := Score(missingProvenance); got != 0 {
		t.Fatalf("Score() with required provenance unsatisfied = %v, want 0", got)
	}
}

func TestFailureModePreservesRankingAndGatePriority(t *testing.T) {
	if got := FailureMode(goncho.RecallBenchmarkCaseReport{ExpectedNoAnswer: true, SelectedMemoryIDs: []string{"mem-1"}}, 0); got != "abstention_failed" {
		t.Fatalf("FailureMode() = %q, want abstention_failed", got)
	}

	lowRank := goncho.RecallBenchmarkCaseReport{
		RelevantIDs:        []string{"mem-3"},
		CandidateMemoryIDs: []string{"mem-1", "mem-2", "mem-3"},
		RecallAt5:          0,
	}
	if got := FailureMode(lowRank, 0); got != "rank_too_low" {
		t.Fatalf("FailureMode() = %q, want rank_too_low", got)
	}

	missingCandidate := lowRank
	missingCandidate.CandidateMemoryIDs = []string{"mem-1", "mem-2"}
	if got := FailureMode(missingCandidate, 0); got != "missing_candidate" {
		t.Fatalf("FailureMode() = %q, want missing_candidate", got)
	}
}

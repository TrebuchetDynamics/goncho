package goncho

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestRecallFactVoiceRanksStructuredEvidence(t *testing.T) {
	now := time.Date(2026, 5, 23, 12, 0, 0, 0, time.UTC)
	engine := newRecallPipelineEngine(staticRecallGenerator{candidates: []RecallCandidate{
		{
			MemoryID:   "mem-lexical-decoy",
			Content:    "Who owns component A-17? This checklist asks who owns component A-17 but does not answer it.",
			ScopeID:    "team",
			CreatedAt:  now,
			Importance: 0.7,
			Provenance: []EvidenceItem{{Kind: "keyword", Score: 1.00, Note: "question-shaped lexical echo"}},
		},
		{
			MemoryID:   "mem-structured-fact-owner",
			Content:    "Nadia owns component A-17.",
			ScopeID:    "team",
			CreatedAt:  now.Add(-time.Hour),
			Importance: 0.7,
			Provenance: []EvidenceItem{
				{Kind: "keyword", Score: 0.35, Note: "partial lexical match"},
				{Kind: "fact", ID: "fact-component-a17-owner", Source: "memoria_fact", Score: 1.00, Note: "subject=component A-17 predicate=owner object=Nadia"},
			},
		},
	}}, recallPipelineOptions{
		pipelineVersion: "fact-voice-test-v1",
		scoringConfig: RecallScoringConfig{
			Version:     "fact-voice-test-v1",
			Weights:     map[string]float64{"keyword": 0.45, "fact": 0.45, "scope": 0.10},
			RRFK:        60,
			MMRLambda:   1,
			TokenBudget: 120,
		},
		now: func() time.Time { return now },
	})

	trace, err := engine.Run(context.Background(), RecallQuery{
		WorkspaceID: "default",
		Peer:        "user-juan",
		Query:       "Who owns component A-17?",
		ScopeID:     "team",
		Limit:       1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(selectedRecallIDs(trace), []string{"mem-structured-fact-owner"}) {
		t.Fatalf("selected IDs = %v, want structured fact owner over lexical decoy", selectedRecallIDs(trace))
	}
	why := strings.Join(trace.Selected[0].Score.WhySelected, "; ")
	if !strings.Contains(why, "fact_score=1.000000") {
		t.Fatalf("why_selected = %q, want fact_score provenance", why)
	}
	report := FormatRecallDiagnosticsReport(BuildRecallDiagnostics(trace))
	if !strings.Contains(report, "fact=1.000000") {
		t.Fatalf("diagnostics report = %q, want fact score surfaced", report)
	}
}

package goncho

import (
	"context"
	"slices"
	"testing"
	"time"
)

func TestRecallTemporalRoutingPrefersCurrentFactAndWarnsOnSupersededEvidence(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	engine := newRecallPipelineEngine(staticRecallGenerator{candidates: []RecallCandidate{
		{
			MemoryID:   "mem-owner-old",
			Content:    "Mira owns component A-17.",
			ScopeID:    "team",
			CreatedAt:  now.Add(-48 * time.Hour),
			Importance: 0.95,
			Provenance: []EvidenceItem{
				{Kind: "keyword", Score: 1.00, Note: "matched component owner"},
				{Kind: "temporal", Score: 0.10, Note: "superseded_by=mem-owner-current"},
			},
		},
		{
			MemoryID:   "mem-owner-current",
			Content:    "Nadia now owns component A-17.",
			ScopeID:    "team",
			CreatedAt:  now.Add(-2 * time.Hour),
			Importance: 0.70,
			Provenance: []EvidenceItem{
				{Kind: "keyword", Score: 0.86, Note: "matched component owner"},
				{Kind: "temporal", Score: 1.00, Note: "current_fact"},
			},
		},
	}}, recallPipelineOptions{
		pipelineVersion: "temporal-routing-test-v1",
		scoringConfig: RecallScoringConfig{
			Version:       "temporal-routing-test-v1",
			Weights:       map[string]float64{"keyword": 0.65, "recency": 0.10, "importance": 0.15, "scope": 0.10},
			RRFK:          60,
			MMRLambda:     0.70,
			DiversityKeys: []string{"memory_id"},
			TokenBudget:   120,
		},
		now: func() time.Time { return now },
	})

	trace, err := engine.Run(context.Background(), RecallQuery{
		WorkspaceID: "default",
		Peer:        "user-juan",
		Query:       "Who owns component A-17 now?",
		ScopeID:     "team",
		Limit:       1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(selectedRecallIDs(trace), []string{"mem-owner-current"}) {
		t.Fatalf("selected IDs = %v, want current owner", selectedRecallIDs(trace))
	}
	if !traceHasWarning(trace, RecallWarningSupersededEvidenceObserved) {
		t.Fatalf("warnings = %+v, want superseded-evidence warning", trace.Warnings)
	}
	if !candidateIDSeen(trace.Candidates, "mem-owner-old") {
		t.Fatalf("candidates = %+v, want superseded evidence preserved", trace.Candidates)
	}
}

func candidateIDSeen(items []ScoredRecallCandidate, memoryID string) bool {
	for _, item := range items {
		if item.Candidate.MemoryID == memoryID {
			return true
		}
	}
	return false
}

package goncho

import (
	"context"
	"slices"
	"testing"
	"time"
)

func TestGraphRecallConnectsOwnerThroughServiceRelation(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	base := staticRecallGenerator{candidates: []RecallCandidate{
		{
			MemoryID:   "mem-auth-service",
			SourceType: "conclusion",
			Content:    "The authentication service handles login, session refresh, and JWT validation.",
			SessionID:  "sess-auth",
			ScopeID:    "team",
			CreatedAt:  now.Add(-2 * time.Hour),
			Importance: 0.80,
			Provenance: []EvidenceItem{{Kind: "keyword", Score: 0.92, Note: "matched authentication service"}},
		},
	}}
	index := GraphExpansionIndex{
		Memories: map[string]RecallCandidate{
			"mem-auth-owner": {
				MemoryID:   "mem-auth-owner",
				SourceType: "conclusion",
				Content:    "Mira is accountable for component A-17 and reviews production incidents for it.",
				SessionID:  "sess-auth",
				ScopeID:    "team",
				CreatedAt:  now.Add(-90 * time.Minute),
				Importance: 0.85,
			},
		},
		Relations: []GraphRelation{
			{
				FromMemoryID: "mem-auth-service",
				ToMemoryID:   "mem-auth-owner",
				Relation:     "owned_by",
				QueryTerms:   []string{"authentication", "owner"},
				EvidenceID:   "edge-auth-owned-by-mira",
				Score:        0.98,
			},
		},
	}
	engine := newRecallPipelineEngine(
		newGraphExpandingRecallGenerator(base, index),
		recallPipelineOptions{
			pipelineVersion: "graph-test-v1",
			scoringConfig: RecallScoringConfig{
				Version:       "graph-test-v1",
				Weights:       map[string]float64{"keyword": 0.30, "graph": 0.60, "scope": 0.10},
				RRFK:          60,
				MMRLambda:     0.70,
				DiversityKeys: []string{"memory_id"},
				TokenBudget:   120,
			},
			now: func() time.Time { return now },
		},
	)

	trace, err := engine.Run(context.Background(), RecallQuery{
		WorkspaceID: "default",
		Peer:        "user-juan",
		Query:       "Who is the owner for the authentication service?",
		ScopeID:     "team",
		Limit:       2,
	})
	if err != nil {
		t.Fatal(err)
	}
	selected := selectedRecallIDs(trace)
	for _, want := range []string{"mem-auth-service", "mem-auth-owner"} {
		if !slices.Contains(selected, want) {
			t.Fatalf("selected IDs = %v, want %q", selected, want)
		}
	}
	owner, ok := selectedRecallCandidate(trace, "mem-auth-owner")
	if !ok {
		t.Fatalf("selected = %+v, want mem-auth-owner", trace.Selected)
	}
	if !candidateHasGraphProvenance(owner, "edge-auth-owned-by-mira") {
		t.Fatalf("owner provenance = %+v, want graph relation path provenance", owner.Provenance)
	}
	if !candidateHasGraphNote(owner, "mem-auth-service -> owned_by -> mem-auth-owner") {
		t.Fatalf("owner provenance = %+v, want relation path provenance", owner.Provenance)
	}
}

func selectedRecallCandidate(trace RecallTrace, memoryID string) (RecallCandidate, bool) {
	for _, selected := range trace.Selected {
		if selected.Candidate.MemoryID == memoryID {
			return selected.Candidate, true
		}
	}
	return RecallCandidate{}, false
}

func candidateHasGraphProvenance(candidate RecallCandidate, evidenceID string) bool {
	for _, item := range candidate.Provenance {
		if item.Kind == "graph" && item.ID == evidenceID {
			return true
		}
	}
	return false
}

func candidateHasGraphNote(candidate RecallCandidate, note string) bool {
	for _, item := range candidate.Provenance {
		if item.Kind == "graph" && item.Note == note {
			return true
		}
	}
	return false
}

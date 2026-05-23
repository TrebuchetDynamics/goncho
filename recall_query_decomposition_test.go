package goncho

import (
	"context"
	"slices"
	"testing"
	"time"
)

func TestRecallQueryDecompositionRetrievesEachSubQuestionFact(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	base := queryKeyedRecallGenerator{candidatesByQuery: map[string][]RecallCandidate{
		"Who owns the authentication service and what incident did that owner review?": {
			{
				MemoryID:   "mem-auth-service",
				Content:    "Authentication service handles login and session refresh.",
				ScopeID:    "team",
				CreatedAt:  now,
				Importance: 0.80,
				Provenance: []EvidenceItem{{Kind: "keyword", Score: 0.88, Note: "matched authentication service"}},
			},
		},
		"Who owns the authentication service?": {
			{
				MemoryID:   "mem-auth-owner",
				Content:    "Mira owns the authentication service.",
				ScopeID:    "team",
				CreatedAt:  now,
				Importance: 0.90,
				Provenance: []EvidenceItem{{Kind: "keyword", Score: 0.95, Note: "matched authentication owner"}},
			},
		},
		"What incident did that owner review?": {
			{
				MemoryID:   "mem-auth-incident",
				Content:    "Mira reviewed incident INC-204 for the authentication service.",
				ScopeID:    "team",
				CreatedAt:  now,
				Importance: 0.85,
				Provenance: []EvidenceItem{{Kind: "keyword", Score: 0.93, Note: "matched owner incident"}},
			},
		},
	}}
	engine := newRecallPipelineEngine(
		newQueryDecomposingRecallGenerator(base, fixedRecallSubqueries(
			"Who owns the authentication service?",
			"What incident did that owner review?",
		)),
		recallPipelineOptions{
			pipelineVersion: "query-decomposition-test-v1",
			scoringConfig: RecallScoringConfig{
				Version:       "query-decomposition-test-v1",
				Weights:       map[string]float64{"keyword": 0.85, "scope": 0.15},
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
		Query:       "Who owns the authentication service and what incident did that owner review?",
		ScopeID:     "team",
		Limit:       3,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"mem-auth-service", "mem-auth-owner", "mem-auth-incident"} {
		if !slices.Contains(selectedRecallIDs(trace), want) {
			t.Fatalf("selected IDs = %v, want decomposed fact %q", selectedRecallIDs(trace), want)
		}
	}
}

func TestRecallQueryDecompositionDeduplicatesStableMemoryIDs(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	base := queryKeyedRecallGenerator{candidatesByQuery: map[string][]RecallCandidate{
		"authentication owner incident": {
			{MemoryID: "mem-auth-owner", Content: "Mira owns authentication.", ScopeID: "team", CreatedAt: now, Importance: 0.80, Provenance: []EvidenceItem{{Kind: "keyword", Score: 0.90}}},
		},
		"authentication owner": {
			{MemoryID: "mem-auth-owner", Content: "Mira owns authentication.", ScopeID: "team", CreatedAt: now, Importance: 0.80, Provenance: []EvidenceItem{{Kind: "keyword", Score: 0.90}}},
		},
		"authentication incident": {
			{MemoryID: "mem-auth-incident", Content: "Mira reviewed INC-204.", ScopeID: "team", CreatedAt: now, Importance: 0.80, Provenance: []EvidenceItem{{Kind: "keyword", Score: 0.90}}},
		},
	}}
	items, err := newQueryDecomposingRecallGenerator(base, fixedRecallSubqueries(
		"authentication owner",
		"authentication incident",
	)).Generate(context.Background(), RecallQuery{Query: "authentication owner incident", ScopeID: "team"})
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.MemoryID)
	}
	if !slices.Equal(ids, []string{"mem-auth-owner", "mem-auth-incident"}) {
		t.Fatalf("merged IDs = %v, want stable memory_id deduplication", ids)
	}
}

type queryKeyedRecallGenerator struct {
	candidatesByQuery map[string][]RecallCandidate
}

func (g queryKeyedRecallGenerator) Generate(ctx context.Context, q RecallQuery) ([]RecallCandidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	items := g.candidatesByQuery[q.Query]
	out := make([]RecallCandidate, len(items))
	copy(out, items)
	return out, nil
}

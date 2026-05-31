package goncho

import (
	"context"
	"slices"
	"testing"
	"time"
)

func TestRecallQueryDecompositionPropagatesBaseGeneratorWarnings(t *testing.T) {
	engine := newRecallPipelineEngine(
		newQueryDecomposingRecallGenerator(staticRecallGenerator{
			warnings: []RecallWarning{{
				Code:     RecallWarningSemanticUnavailable,
				Stage:    RecallStageGenerate,
				Severity: RecallWarningDegraded,
				Message:  "semantic provider unavailable during decomposed recall",
			}},
		}, fixedRecallSubqueries("authentication owner")),
		recallPipelineOptions{scoringConfig: RecallScoringConfig{Version: "warning-propagation-test-v1"}},
	)

	trace, err := engine.Run(context.Background(), RecallQuery{Query: "authentication incident", ScopeID: "team"})
	if err != nil {
		t.Fatal(err)
	}
	if !recallTraceHasWarning(trace, RecallWarningSemanticUnavailable) {
		t.Fatalf("trace warnings = %+v, want wrapped base generator warning", trace.Warnings)
	}
}

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

func TestRecallQueryDecompositionSkipsTrimEquivalentSubqueries(t *testing.T) {
	base := &recordingRecallGenerator{}
	_, err := newQueryDecomposingRecallGenerator(base, fixedRecallSubqueries(
		"authentication owner",
		"  authentication owner  ",
	)).Generate(context.Background(), RecallQuery{Query: " authentication owner ", ScopeID: "team"})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(base.queries, []string{" authentication owner "}) {
		t.Fatalf("generated queries = %q, want only original query when subqueries are trim-equivalent", base.queries)
	}
}

func TestPlannedRecallQueriesKeepsRetrievalDistinctVariants(t *testing.T) {
	queries := plannedRecallQueries(
		RecallQuery{Query: "authentication owner", ScopeID: "team", Sources: []string{"memory"}},
		func(q RecallQuery) []RecallQuery {
			return []RecallQuery{{Query: " authentication owner ", ScopeID: "team", Sources: []string{"session"}}}
		},
	)
	if got := len(queries); got != 2 {
		t.Fatalf("planned query count = %d, want original plus retrieval-distinct same-text subquery: %+v", got, queries)
	}
	if !slices.Equal(queries[0].Sources, []string{"memory"}) || !slices.Equal(queries[1].Sources, []string{"session"}) {
		t.Fatalf("planned query sources = %v then %v, want both source filters retained", queries[0].Sources, queries[1].Sources)
	}
}

func TestPlannedRecallQueriesDeduplicatesWildcardSourceVariants(t *testing.T) {
	queries := plannedRecallQueries(
		RecallQuery{Query: "authentication owner"},
		func(q RecallQuery) []RecallQuery {
			return []RecallQuery{{Query: " authentication owner ", Sources: []string{"*"}}}
		},
	)
	if got := len(queries); got != 1 {
		t.Fatalf("planned query count = %d, want empty and wildcard source filters treated as equivalent: %+v", got, queries)
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

func TestMergeRecallCandidateEvidenceKeepsMetadataWhenScoreImproves(t *testing.T) {
	merged := mergeRecallCandidateEvidence(
		RecallCandidate{MemoryID: "mem-auth-owner", Provenance: []EvidenceItem{{Kind: "fact", Source: "annotations", ID: "ann-owner", Note: "owner fact", Score: 0.40, Metadata: map[string]string{"fact_type": "ownership"}}}},
		RecallCandidate{MemoryID: "mem-auth-owner", Provenance: []EvidenceItem{{Kind: "fact", Source: "annotations", ID: "ann-owner", Note: "owner fact", Score: 0.95}}},
	)
	if len(merged.Provenance) != 1 {
		t.Fatalf("merged provenance count = %d, want one evidence item", len(merged.Provenance))
	}
	if got := merged.Provenance[0].Score; got != 0.95 {
		t.Fatalf("merged evidence score = %v, want improved score", got)
	}
	if got := merged.Provenance[0].Metadata["fact_type"]; got != "ownership" {
		t.Fatalf("merged evidence metadata = %+v, want existing provenance metadata preserved", merged.Provenance[0].Metadata)
	}
}

func TestMergeRecallCandidateEvidenceKeepsStrongestImportance(t *testing.T) {
	merged := mergeRecallCandidateEvidence(
		RecallCandidate{MemoryID: "mem-auth-owner", Importance: 0.20, Provenance: []EvidenceItem{{Kind: "keyword", Score: 0.90, Note: "original query"}}},
		RecallCandidate{MemoryID: "mem-auth-owner", Importance: 0.95, Provenance: []EvidenceItem{{Kind: "fact", Score: 1.00, Note: "decomposed owner fact"}}},
	)
	if got := merged.Importance; got != 0.95 {
		t.Fatalf("merged importance = %v, want strongest duplicate candidate importance", got)
	}
}

func TestRecallQueryDecompositionMergesProvenanceForDuplicateMemoryIDs(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	base := queryKeyedRecallGenerator{candidatesByQuery: map[string][]RecallCandidate{
		"authentication owner incident": {
			{MemoryID: "mem-auth-owner", Content: "Mira owns authentication.", ScopeID: "team", CreatedAt: now, Importance: 0.80, Provenance: []EvidenceItem{{Kind: "keyword", Source: "fts", Score: 0.40, Note: "original composite query"}}},
		},
		"authentication owner": {
			{MemoryID: "mem-auth-owner", Content: "Mira owns authentication.", ScopeID: "team", CreatedAt: now, Importance: 0.80, Provenance: []EvidenceItem{{Kind: "fact", Source: "goncho_memory_annotations", ID: "ann-owner", Score: 1.0, Note: "fact=Mira owns authentication"}}},
		},
	}}
	items, err := newQueryDecomposingRecallGenerator(base, fixedRecallSubqueries(
		"authentication owner",
	)).Generate(context.Background(), RecallQuery{Query: "authentication owner incident", ScopeID: "team"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].MemoryID != "mem-auth-owner" {
		t.Fatalf("merged candidates = %+v, want one stable memory candidate", items)
	}
	if len(items[0].Provenance) != 2 {
		t.Fatalf("merged provenance = %+v, want original and decomposed-query evidence retained", items[0].Provenance)
	}
	if !evidenceListHas(items[0].Provenance, "fact", "ann-owner") {
		t.Fatalf("merged provenance = %+v, want decomposed fact evidence retained", items[0].Provenance)
	}
}

type queryKeyedRecallGenerator struct {
	candidatesByQuery map[string][]RecallCandidate
}

type recordingRecallGenerator struct {
	queries []string
}

func (g *recordingRecallGenerator) Generate(ctx context.Context, q RecallQuery) ([]RecallCandidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	g.queries = append(g.queries, q.Query)
	return nil, nil
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

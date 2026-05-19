package goncho

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestRecallTraceStableIDAndJSONFixture(t *testing.T) {
	now := time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)
	query := RecallQuery{
		WorkspaceID: "default",
		Peer:        "user-juan",
		Query:       "auth rate limit",
		SessionKey:  "sess-auth",
		ScopeID:     "team",
		Limit:       2,
	}
	config := RecallScoringConfig{
		Version:       "test-v1",
		Weights:       map[string]float64{"keyword": 0.25, "semantic": 0.30, "graph": 0.20, "recency": 0.10, "importance": 0.10, "scope": 0.05},
		RRFK:          60,
		MMRLambda:     0.70,
		DiversityKeys: []string{"session_id"},
		TokenBudget:   80,
	}
	engine := newRecallPipelineEngine(staticRecallGenerator{candidates: []RecallCandidate{
		{
			MemoryID:   "mem-auth",
			SourceType: "conclusion",
			Content:    "JWT auth uses jose middleware.",
			SessionID:  "sess-auth",
			AgentID:    "gormes",
			ScopeID:    "team",
			CreatedAt:  now.Add(-time.Hour),
			Importance: 0.90,
			Provenance: []EvidenceItem{
				{Kind: "keyword", Score: 0.80, Note: "matched auth"},
				{Kind: "semantic", Score: 0.90, Note: "embedding neighbor"},
				{Kind: "graph", Score: 0.50, Note: "AUTH_USES edge"},
				{Kind: "scope", Score: 1.00, Note: "same scope"},
			},
		},
		{
			MemoryID:   "mem-rate",
			SourceType: "turn",
			Content:    "Rate limiting uses token bucket middleware.",
			SessionID:  "sess-auth",
			AgentID:    "gormes",
			ScopeID:    "team",
			CreatedAt:  now.Add(-2 * time.Hour),
			Importance: 0.70,
			Provenance: []EvidenceItem{
				{Kind: "keyword", Score: 0.70, Note: "matched rate limit"},
				{Kind: "semantic", Score: 0.86, Note: "embedding neighbor"},
				{Kind: "graph", Score: 0.30, Note: "related middleware edge"},
				{Kind: "scope", Score: 1.00, Note: "same scope"},
			},
		},
		{
			MemoryID:   "mem-db",
			SourceType: "conclusion",
			Content:    "Database performance work found an N+1 query.",
			SessionID:  "sess-db",
			AgentID:    "gormes",
			ScopeID:    "team",
			CreatedAt:  now.Add(-30 * time.Minute),
			Importance: 0.60,
			Provenance: []EvidenceItem{
				{Kind: "keyword", Score: 0.20, Note: "weak lexical overlap"},
				{Kind: "semantic", Score: 0.30, Note: "weak embedding neighbor"},
				{Kind: "graph", Score: 0.95, Note: "database performance edge"},
				{Kind: "scope", Score: 1.00, Note: "same scope"},
			},
		},
	}}, recallPipelineOptions{
		pipelineVersion: "test-pipeline",
		scoringConfig:   config,
		now:             func() time.Time { return now },
	})

	trace, err := engine.Run(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	again, err := engine.Run(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	if trace.TraceID == "" {
		t.Fatal("TraceID is empty")
	}
	if trace.TraceID != again.TraceID {
		t.Fatalf("TraceID changed across identical runs: %q vs %q", trace.TraceID, again.TraceID)
	}
	if !slices.Equal(selectedRecallIDs(trace), []string{"mem-auth", "mem-db"}) {
		t.Fatalf("selected IDs = %v, want diversity to pick mem-auth then mem-db", selectedRecallIDs(trace))
	}

	raw, err := trace.StableJSON()
	if err != nil {
		t.Fatal(err)
	}
	golden := filepath.Join("testdata", "recall_trace", "stable_trace.golden.json")
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != string(want) {
		t.Fatalf("stable trace JSON mismatch\nwant:\n%s\n\ngot:\n%s", want, raw)
	}
}

func TestRecallProjectionIsTraceOnly(t *testing.T) {
	projectorType := reflect.TypeOf(&RecallProjector{})
	if projectorType.NumMethod() == 0 {
		t.Fatal("RecallProjector exposes no methods")
	}
	recallTraceType := reflect.TypeOf(RecallTrace{})
	candidateSliceType := reflect.TypeOf([]RecallCandidate{})
	for i := 0; i < projectorType.NumMethod(); i++ {
		method := projectorType.Method(i)
		if method.PkgPath != "" {
			continue
		}
		if method.Type.NumIn() < 2 || method.Type.In(1) != recallTraceType {
			t.Fatalf("projector method %s must accept RecallTrace as its only projection input, got %s", method.Name, method.Type)
		}
		for j := 1; j < method.Type.NumIn(); j++ {
			if method.Type.In(j) == candidateSliceType {
				t.Fatalf("projector method %s accepts raw candidates", method.Name)
			}
		}
	}

	trace := RecallTrace{
		Query: RecallQuery{WorkspaceID: "default", Peer: "user-juan", Query: "auth"},
		Selected: []ScoredRecallCandidate{{
			Candidate: RecallCandidate{
				MemoryID:   "42",
				SourceType: "conclusion",
				Content:    "JWT auth uses jose.",
				SessionID:  "sess-auth",
			},
			Score: RecallScore{FinalScore: 0.91, WhySelected: []string{"highest final_score"}},
		}},
	}
	search := (&RecallProjector{}).ProjectSearch(trace)
	if search.WorkspaceID != "default" || search.Peer != "user-juan" || search.Query != "auth" {
		t.Fatalf("projected search metadata = %+v", search)
	}
	if len(search.Results) != 1 || search.Results[0].Content != "JWT auth uses jose." {
		t.Fatalf("projected search results = %+v", search.Results)
	}
	context := (&RecallProjector{}).ProjectContext(trace)
	if !strings.Contains(context.Representation, "JWT auth uses jose.") {
		t.Fatalf("projected context representation = %q", context.Representation)
	}
}

func selectedRecallIDs(trace RecallTrace) []string {
	ids := make([]string, 0, len(trace.Selected))
	for _, item := range trace.Selected {
		ids = append(ids, item.Candidate.MemoryID)
	}
	return ids
}

package goncho

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/goncho/memory"
)

func TestSearchCandidateGenerationKeepsOldStrongLexicalMatch(t *testing.T) {
	svc, ctx := newCandidateGenerationService(t)
	insertOldStrongLexicalMatchWithDistractors(t, ctx, svc, 650)

	got, err := svc.Search(ctx, SearchParams{Peer: "peer", Query: "rare orchid retrieval marker", Limit: 10, MaxTokens: 100_000})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(got.Results) == 0 || got.Results[0].SessionKey != "old" {
		t.Fatalf("top result = %+v, want old strong lexical candidate to survive pre-rank candidate generation", got.Results)
	}
}

func TestSearchCandidateGenerationScansBeyondFiveThousandRecentDistractors(t *testing.T) {
	svc, ctx := newCandidateGenerationService(t)
	insertOldStrongLexicalMatchWithDistractors(t, ctx, svc, 5001)

	got, err := svc.Search(ctx, SearchParams{Peer: "peer", Query: "rare orchid retrieval marker", Limit: 10, MaxTokens: 100_000})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(got.Results) == 0 || got.Results[0].SessionKey != "old" {
		t.Fatalf("top result = %+v, want old strong lexical candidate beyond the fixed scan window to survive pre-rank candidate generation", got.Results)
	}
}

func TestPlanConclusionCandidateScanExposesRankBeforeTopKInvariant(t *testing.T) {
	queried := planConclusionCandidateScan(" rare orchid ", 10)
	if queried.TrimmedQuery != "rare orchid" || queried.Bounded {
		t.Fatalf("queried scan plan = %+v, want unbounded pre-rank candidate scan", queried)
	}

	browsing := planConclusionCandidateScan(" ", 10)
	if browsing.TrimmedQuery != "" || !browsing.Bounded || browsing.Limit != 10 {
		t.Fatalf("empty-query scan plan = %+v, want bounded recency scan", browsing)
	}
}

func newCandidateGenerationService(t *testing.T) (*Service, context.Context) {
	t.Helper()
	ctx := context.Background()
	store, err := memory.OpenSqlite(filepath.Join(t.TempDir(), "candidate.db"), 0, nil)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = store.Close(ctx) })
	if err := RunMigrations(store.DB()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewService(store.DB(), Config{WorkspaceID: "candidate-test", ObserverPeerID: "observer", RecentMessages: 0}, nil), ctx
}

func insertOldStrongLexicalMatchWithDistractors(t *testing.T, ctx context.Context, svc *Service, distractors int) {
	t.Helper()
	if _, err := svc.Conclude(ctx, ConcludeParams{Peer: "peer", SessionKey: "old", Scope: "benchmark", Conclusion: "Maya stores the rare orchid retrieval marker in the archive cabinet."}); err != nil {
		t.Fatalf("insert old gold: %v", err)
	}
	for i := 0; i < distractors; i++ {
		if _, err := svc.Conclude(ctx, ConcludeParams{Peer: "peer", SessionKey: fmt.Sprintf("new-%04d", i), Scope: "benchmark", Conclusion: fmt.Sprintf("Recent distractor %04d about dashboards and notes.", i)}); err != nil {
			t.Fatalf("insert distractor %d: %v", i, err)
		}
	}
}

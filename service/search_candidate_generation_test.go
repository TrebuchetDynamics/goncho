package goncho

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/goncho/memory"
)

func TestSearchCandidateGenerationKeepsOldStrongLexicalMatch(t *testing.T) {
	ctx := context.Background()
	store, err := memory.OpenSqlite(filepath.Join(t.TempDir(), "candidate.db"), 0, nil)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer store.Close(ctx)
	if err := RunMigrations(store.DB()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	svc := NewService(store.DB(), Config{WorkspaceID: "candidate-test", ObserverPeerID: "observer", RecentMessages: 0}, nil)

	if _, err := svc.Conclude(ctx, ConcludeParams{Peer: "peer", SessionKey: "old", Scope: "benchmark", Conclusion: "Maya stores the rare orchid retrieval marker in the archive cabinet."}); err != nil {
		t.Fatalf("insert old gold: %v", err)
	}
	for i := 0; i < 650; i++ {
		if _, err := svc.Conclude(ctx, ConcludeParams{Peer: "peer", SessionKey: fmt.Sprintf("new-%03d", i), Scope: "benchmark", Conclusion: fmt.Sprintf("Recent distractor %03d about dashboards and notes.", i)}); err != nil {
			t.Fatalf("insert distractor %d: %v", i, err)
		}
	}

	got, err := svc.Search(ctx, SearchParams{Peer: "peer", Query: "rare orchid retrieval marker", Limit: 10, MaxTokens: 100_000})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(got.Results) == 0 || got.Results[0].SessionKey != "old" {
		t.Fatalf("top result = %+v, want old strong lexical candidate to survive pre-rank candidate generation", got.Results)
	}
}

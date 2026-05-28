package goncho

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/goncho/memory"
)

func TestReindexPreviewReportsConclusionCountsWithoutMutatingOrEmbedding(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "reindex-preview.db")
	store, err := memory.OpenSqlite(dbPath, 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(ctx)
	if err := RunMigrations(store.DB()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	workspace := "reindex-workspace"
	peer := "user-reindex"
	session := "reindex-session"

	indexPath := filepath.Join(t.TempDir(), "vectors.json")
	provider := fakeTextEmbeddingProvider{dims: 4}
	index, err := NewLocalVectorIndex(ctx, LocalVectorIndexOptions{Path: indexPath, Provider: provider})
	if err != nil {
		t.Fatalf("NewLocalVectorIndex: %v", err)
	}

	svc := NewService(store.DB(), Config{
		WorkspaceID:    workspace,
		ObserverPeerID: "assistant",
		VectorStore:    index,
	}, nil)

	// Seed three conclusions.
	conclusions := []string{
		"First conclusion already indexed.",
		"Second conclusion not in vector index.",
		"Third conclusion fresh and matching.",
	}
	for _, c := range conclusions {
		if _, err := svc.Conclude(ctx, ConcludeParams{Peer: peer, SessionKey: session, Conclusion: c}); err != nil {
			t.Fatalf("Conclude %q: %v", c, err)
		}
	}

	// Manually index just the first conclusion.
	if err := index.Upsert(ctx, LocalVectorMemory{
		MemoryID:    "1",
		WorkspaceID: workspace,
		Peer:        peer,
		SourceType:  "conclusion",
		Content:     conclusions[0],
		SessionID:   session,
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	result, err := svc.ReindexPreview(ctx)
	if err != nil {
		t.Fatalf("ReindexPreview: %v", err)
	}
	if result.Mutates {
		t.Fatalf("ReindexPreview mutates = true, want false")
	}
	if result.Status != "ok" {
		t.Fatalf("ReindexPreview status = %q, want ok", result.Status)
	}
	if result.Total != 3 {
		t.Fatalf("ReindexPreview total = %d, want 3", result.Total)
	}
	if result.NotIndexed != 2 {
		t.Fatalf("ReindexPreview not_indexed = %d, want 2", result.NotIndexed)
	}
	if result.Fresh != 1 {
		t.Fatalf("ReindexPreview fresh = %d, want 1", result.Fresh)
	}
	if result.Stale != 0 {
		t.Fatalf("ReindexPreview stale = %d, want 0", result.Stale)
	}

	// Re-run to confirm deterministic.
	again, err := svc.ReindexPreview(ctx)
	if err != nil {
		t.Fatalf("ReindexPreview again: %v", err)
	}
	if again.Total != result.Total || again.NotIndexed != result.NotIndexed || again.Fresh != result.Fresh {
		t.Fatalf("ReindexPreview not deterministic: first=%+v second=%+v", result, again)
	}
}

func TestReindexPreviewDetectsStaleRowsWhenContentChanges(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "reindex-stale.db")
	store, err := memory.OpenSqlite(dbPath, 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(ctx)
	if err := RunMigrations(store.DB()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	workspace := "stale-workspace"
	peer := "user-stale"

	indexPath := filepath.Join(t.TempDir(), "stale-vectors.json")
	provider := fakeTextEmbeddingProvider{dims: 4}
	index, err := NewLocalVectorIndex(ctx, LocalVectorIndexOptions{Path: indexPath, Provider: provider})
	if err != nil {
		t.Fatalf("NewLocalVectorIndex: %v", err)
	}

	svc := NewService(store.DB(), Config{
		WorkspaceID:    workspace,
		ObserverPeerID: "assistant",
		VectorStore:    index,
	}, nil)

	// Seed a conclusion, then manually index with OLD content.
	if _, err := svc.Conclude(ctx, ConcludeParams{Peer: peer, Conclusion: "The current version of this conclusion."}); err != nil {
		t.Fatalf("Conclude: %v", err)
	}
	if err := index.Upsert(ctx, LocalVectorMemory{
		MemoryID:    "1",
		WorkspaceID: workspace,
		Peer:        peer,
		Content:     "An older version of this conclusion.",
	}); err != nil {
		t.Fatalf("Upsert old content: %v", err)
	}

	result, err := svc.ReindexPreview(ctx)
	if err != nil {
		t.Fatalf("ReindexPreview: %v", err)
	}
	if result.Mutates {
		t.Fatalf("mutates = true")
	}
	if result.Total != 1 {
		t.Fatalf("total = %d, want 1", result.Total)
	}
	if result.Stale != 1 {
		t.Fatalf("stale = %d, want 1", result.Stale)
	}
	if result.Fresh != 0 {
		t.Fatalf("fresh = %d, want 0", result.Fresh)
	}
}

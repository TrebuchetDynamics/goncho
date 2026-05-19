package goncho

import (
	"context"
	"testing"
)

func TestRetrieve_FTSSearch(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	StoreMemory(ctx, db, StoreParams{Content: "User prefers SQLite for local development", Kind: KindPreference, PeerID: "p1", WorkspaceID: "w1"})
	StoreMemory(ctx, db, StoreParams{Content: "Postgres is better for production workloads", Kind: KindFact, PeerID: "p1", WorkspaceID: "w1"})
	StoreMemory(ctx, db, StoreParams{Content: "User likes Go programming language", Kind: KindPreference, PeerID: "p1", WorkspaceID: "w1"})

	result, err := Retrieve(ctx, db, RetrieveParams{
		Query:       "SQLite",
		PeerID:      "p1",
		WorkspaceID: "w1",
		Limit:       5,
	})
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if len(result.Memories) == 0 {
		t.Fatal("expected memories for SQLite query")
	}
	if result.Memories[0].Content != "User prefers SQLite for local development" {
		t.Fatalf("top result = %q, want SQLite preference", result.Memories[0].Content)
	}
	if result.Trace.FTSHits == 0 {
		t.Fatal("expected FTS hits")
	}
}

func TestRetrieve_FiltersByKind(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	StoreMemory(ctx, db, StoreParams{Content: "Preference: SQLite", Kind: KindPreference, PeerID: "p1", WorkspaceID: "w1"})
	StoreMemory(ctx, db, StoreParams{Content: "Fact: Go is fast", Kind: KindFact, PeerID: "p1", WorkspaceID: "w1"})

	result, err := Retrieve(ctx, db, RetrieveParams{
		Query:       "",
		PeerID:      "p1",
		WorkspaceID: "w1",
		Kinds:       []Kind{KindPreference},
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	for _, m := range result.Memories {
		if m.Kind != KindPreference {
			t.Fatalf("got kind %q, want only preferences", m.Kind)
		}
	}
}

func TestRetrieve_GraphExpansion(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	StoreMemory(ctx, db, StoreParams{Content: "Alice prefers PostgreSQL", Kind: KindFact, PeerID: "p1", WorkspaceID: "w1"})
	StoreMemory(ctx, db, StoreParams{Content: "PostgreSQL handles JSON well", Kind: KindFact, PeerID: "p1", WorkspaceID: "w1"})

	result, err := Retrieve(ctx, db, RetrieveParams{
		Query:       "Alice",
		PeerID:      "p1",
		WorkspaceID: "w1",
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if result.Trace.GraphHits > 0 {
		t.Logf("graph expanded %d additional memories", result.Trace.GraphHits)
	}
}

func TestRetrieve_ContextBoost(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	StoreMemory(ctx, db, StoreParams{Content: "Project X uses SQLite", Kind: KindFact, PeerID: "p1", WorkspaceID: "w1", ContextID: "project-x"})
	StoreMemory(ctx, db, StoreParams{Content: "Old project used Postgres", Kind: KindFact, PeerID: "p1", WorkspaceID: "w1", ContextID: "old-project"})

	result, err := Retrieve(ctx, db, RetrieveParams{
		Query:       "database",
		PeerID:      "p1",
		WorkspaceID: "w1",
		ContextID:   "project-x",
		Limit:       5,
	})
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if len(result.Memories) > 0 && result.Memories[0].ContextID != "project-x" {
		t.Logf("top result context = %q, expected project-x boost", result.Memories[0].ContextID)
	}
}

func TestRetrieve_ExcludesForgotten(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	s1, _ := StoreMemory(ctx, db, StoreParams{Content: "Forgotten fact", Kind: KindFact, PeerID: "p1", WorkspaceID: "w1"})
	StoreMemory(ctx, db, StoreParams{Content: "Active fact", Kind: KindFact, PeerID: "p1", WorkspaceID: "w1"})

	ForgetMemory(ctx, db, s1.Memory.ID, ForgetParams{})

	result, err := Retrieve(ctx, db, RetrieveParams{
		Query:       "fact",
		PeerID:      "p1",
		WorkspaceID: "w1",
		Limit:       10,
	})
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	for _, m := range result.Memories {
		if m.ID == s1.Memory.ID {
			t.Fatal("forgotten memory should not appear in results")
		}
	}
}

func TestRetrieve_EmptyQuery(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	StoreMemory(ctx, db, StoreParams{Content: "Test memory one", Kind: KindFact, PeerID: "p1", WorkspaceID: "w1"})
	StoreMemory(ctx, db, StoreParams{Content: "Test memory two", Kind: KindFact, PeerID: "p1", WorkspaceID: "w1"})

	result, err := Retrieve(ctx, db, RetrieveParams{
		Query:       "",
		PeerID:      "p1",
		WorkspaceID: "w1",
		Limit:       5,
	})
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if len(result.Memories) == 0 {
		t.Fatal("expected memories with empty query")
	}
}

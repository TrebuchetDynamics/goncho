package goncho

import (
	"context"
	"testing"

)

// TestWorkspaceIsolation_ScopedMemories proves memories are filtered by
// workspace_id by default.
func TestWorkspaceIsolation_ScopedMemories(t *testing.T) {
	store, err := OpenTestSqlite(t.TempDir()+"/memory.db")
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(context.Background())

	ctx := context.Background()
	svcA := NewService(store.DB(), Config{
		WorkspaceID:    "workspace-a",
		ObserverPeerID: "gormes",
		RecentMessages: 4,
	}, nil)
	svcB := NewService(store.DB(), Config{
		WorkspaceID:    "workspace-b",
		ObserverPeerID: "gormes",
		RecentMessages: 4,
	}, nil)

	// Write a workspace-scoped conclusion in workspace A
	_, err = svcA.Conclude(ctx, ConcludeParams{
		Peer:       "alice",
		Conclusion: "Alice prefers TypeScript over JavaScript",
		Scope:      "workspace",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Search in workspace A should find it
	resultsA, err := svcA.Search(ctx, SearchParams{
		Peer:  "alice",
		Query: "TypeScript",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resultsA.Results) != 1 {
		t.Fatalf("workspace A search: got %d results, want 1", len(resultsA.Results))
	}

	// Search in workspace B should NOT find workspace-A scoped conclusion
	resultsB, err := svcB.Search(ctx, SearchParams{
		Peer:  "alice",
		Query: "TypeScript",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resultsB.Results) != 0 {
		t.Fatalf("workspace B search: got %d results, want 0 (workspace-A scoped memory should not leak)", len(resultsB.Results))
	}
}

// TestWorkspaceIsolation_GlobalMemoriesCrossBoundaries proves global-scope
// memories are visible across workspaces.
func TestWorkspaceIsolation_GlobalMemoriesCrossBoundaries(t *testing.T) {
	store, err := OpenTestSqlite(t.TempDir()+"/memory.db")
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(context.Background())

	ctx := context.Background()
	svcA := NewService(store.DB(), Config{
		WorkspaceID:    "workspace-a",
		ObserverPeerID: "gormes",
		RecentMessages: 4,
	}, nil)
	svcB := NewService(store.DB(), Config{
		WorkspaceID:    "workspace-b",
		ObserverPeerID: "gormes",
		RecentMessages: 4,
	}, nil)

	// Write a global-scoped conclusion in workspace A
	_, err = svcA.Conclude(ctx, ConcludeParams{
		Peer:       "bob",
		Conclusion: "Always use HTTPS for external API calls",
		Scope:      "global",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Search in workspace A should find it
	resultsA, err := svcA.Search(ctx, SearchParams{
		Peer:  "bob",
		Query: "HTTPS",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resultsA.Results) != 1 {
		t.Fatalf("workspace A search: got %d results, want 1", len(resultsA.Results))
	}

	// Search in workspace B should ALSO find global-scoped conclusion
	resultsB, err := svcB.Search(ctx, SearchParams{
		Peer:  "bob",
		Query: "HTTPS",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resultsB.Results) != 1 {
		t.Fatalf("workspace B search: got %d results, want 1 (global memory should cross boundaries)", len(resultsB.Results))
	}
	if resultsB.Results[0].Content != "Always use HTTPS for external API calls" {
		t.Fatalf("workspace B search content = %q, want %q", resultsB.Results[0].Content, "Always use HTTPS for external API calls")
	}
}

// TestWorkspaceIsolation_NoCrossPollution proves workspace-A memories never
// leak into workspace-B context.
func TestWorkspaceIsolation_NoCrossPollution(t *testing.T) {
	store, err := OpenTestSqlite(t.TempDir()+"/memory.db")
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(context.Background())

	ctx := context.Background()
	svcA := NewService(store.DB(), Config{
		WorkspaceID:    "project-alpha",
		ObserverPeerID: "gormes",
		RecentMessages: 4,
	}, nil)
	svcB := NewService(store.DB(), Config{
		WorkspaceID:    "project-beta",
		ObserverPeerID: "gormes",
		RecentMessages: 4,
	}, nil)

	// Write multiple workspace-scoped conclusions in workspace A
	conclusions := []string{
		"Alpha uses React for frontend",
		"Alpha API runs on port 3000",
		"Alpha database is PostgreSQL",
	}
	for _, c := range conclusions {
		_, err := svcA.Conclude(ctx, ConcludeParams{
			Peer:       "team-lead",
			Conclusion: c,
			Scope:      "workspace",
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	// Write different workspace-scoped conclusions in workspace B
	betaConclusions := []string{
		"Beta uses Vue for frontend",
		"Beta API runs on port 8080",
		"Beta database is MongoDB",
	}
	for _, c := range betaConclusions {
		_, err := svcB.Conclude(ctx, ConcludeParams{
			Peer:       "team-lead",
			Conclusion: c,
			Scope:      "workspace",
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	// Search in workspace B should only see beta conclusions
	resultsB, err := svcB.Search(ctx, SearchParams{
		Peer:  "team-lead",
		Query: "frontend",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resultsB.Results) != 1 {
		t.Fatalf("workspace B search: got %d results, want 1", len(resultsB.Results))
	}
	if resultsB.Results[0].Content != "Beta uses Vue for frontend" {
		t.Fatalf("workspace B search content = %q, want %q", resultsB.Results[0].Content, "Beta uses Vue for frontend")
	}

	// Search in workspace A should only see alpha conclusions
	resultsA, err := svcA.Search(ctx, SearchParams{
		Peer:  "team-lead",
		Query: "frontend",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resultsA.Results) != 1 {
		t.Fatalf("workspace A search: got %d results, want 1", len(resultsA.Results))
	}
	if resultsA.Results[0].Content != "Alpha uses React for frontend" {
		t.Fatalf("workspace A search content = %q, want %q", resultsA.Results[0].Content, "Alpha uses React for frontend")
	}

	// Verify no alpha conclusions appear in workspace B search with broader query
	allResultsB, err := svcB.Search(ctx, SearchParams{
		Peer:  "team-lead",
		Query: "",
		Limit: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, hit := range allResultsB.Results {
		for _, alpha := range conclusions {
			if hit.Content == alpha {
				t.Fatalf("workspace B search leaked workspace-A conclusion: %q", alpha)
			}
		}
	}
}

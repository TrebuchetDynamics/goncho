package goncho

import (
	"context"
	"testing"

	"github.com/TrebuchetDynamics/goncho/memory"
)

func TestGormesMultiProfilePeerCardsAreIsolated(t *testing.T) {
	ctx := context.Background()
	store, err := memory.OpenSqlite(t.TempDir()+"/memory.db", 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(ctx)
	if err := RunMigrations(store.DB()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	svc := NewService(store.DB(), Config{WorkspaceID: "gormes", ObserverPeerID: "gormes-runtime"}, nil)
	peer := "telegram:6586915095"
	if err := svc.SetProfileInNamespace(ctx, MemoryNamespace{WorkspaceID: "gormes", ProfileID: "mineru", PeerID: peer, Scope: MemoryScopeProfile}, []string{"Mineru profile memory"}); err != nil {
		t.Fatalf("set mineru profile: %v", err)
	}
	if err := svc.SetProfileInNamespace(ctx, MemoryNamespace{WorkspaceID: "gormes", ProfileID: "yunobo", PeerID: peer, Scope: MemoryScopeProfile}, []string{"Yunobo profile memory"}); err != nil {
		t.Fatalf("set yunobo profile: %v", err)
	}

	mineru, err := svc.ProfileInNamespace(ctx, MemoryNamespace{WorkspaceID: "gormes", ProfileID: "mineru", PeerID: peer, Scope: MemoryScopeProfile})
	if err != nil {
		t.Fatalf("mineru profile: %v", err)
	}
	if mineru.ProfileID != "mineru" || len(mineru.Card) != 1 || mineru.Card[0] != "Mineru profile memory" {
		t.Fatalf("mineru profile = %+v", mineru)
	}
	yunobo, err := svc.ProfileInNamespace(ctx, MemoryNamespace{WorkspaceID: "gormes", ProfileID: "yunobo", PeerID: peer, Scope: MemoryScopeProfile})
	if err != nil {
		t.Fatalf("yunobo profile: %v", err)
	}
	if yunobo.ProfileID != "yunobo" || len(yunobo.Card) != 1 || yunobo.Card[0] != "Yunobo profile memory" {
		t.Fatalf("yunobo profile = %+v", yunobo)
	}
}

func TestGormesMultiProfileMemoryIsolation(t *testing.T) {
	ctx := context.Background()
	store, err := memory.OpenSqlite(t.TempDir()+"/memory.db", 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(ctx)
	if err := RunMigrations(store.DB()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	svc := NewService(store.DB(), Config{WorkspaceID: "gormes", ObserverPeerID: "gormes-runtime"}, nil)

	_, err = svc.Conclude(ctx, ConcludeParams{
		ProfileID:  "mineru",
		Peer:       "telegram:6586915095",
		Conclusion: "Mineru owns fleet reliability memory.",
	})
	if err != nil {
		t.Fatalf("mineru conclude: %v", err)
	}
	_, err = svc.Conclude(ctx, ConcludeParams{
		ProfileID:  "yunobo",
		Peer:       "telegram:6586915095",
		Conclusion: "Yunobo owns trading bot memory.",
	})
	if err != nil {
		t.Fatalf("yunobo conclude: %v", err)
	}

	mineru, err := svc.Search(ctx, SearchParams{
		ProfileID: "mineru",
		Peer:      "telegram:6586915095",
		Query:     "owns memory",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("mineru search: %v", err)
	}
	if mineru.ProfileID != "mineru" {
		t.Fatalf("mineru result profile_id = %q, want mineru", mineru.ProfileID)
	}
	if len(mineru.Results) != 1 || mineru.Results[0].Content != "Mineru owns fleet reliability memory." {
		t.Fatalf("mineru results = %+v, want only mineru memory", mineru.Results)
	}
	if mineru.ScopeEvidence == nil || mineru.ScopeEvidence.Decision != CrossChatDecisionAllowed {
		t.Fatalf("mineru scope evidence = %+v, want allowed profile isolation evidence", mineru.ScopeEvidence)
	}
	if mineru.ScopeEvidence.Scope != MemoryScopeProfile {
		t.Fatalf("mineru scope = %q, want %q", mineru.ScopeEvidence.Scope, MemoryScopeProfile)
	}

	yunobo, err := svc.Context(ctx, ContextParams{
		ProfileID: "yunobo",
		Peer:      "telegram:6586915095",
		Query:     "owns memory",
		MaxTokens: 200,
	})
	if err != nil {
		t.Fatalf("yunobo context: %v", err)
	}
	if yunobo.ProfileID != "yunobo" {
		t.Fatalf("yunobo context profile_id = %q, want yunobo", yunobo.ProfileID)
	}
	if len(yunobo.Conclusions) != 1 || yunobo.Conclusions[0] != "Yunobo owns trading bot memory." {
		t.Fatalf("yunobo conclusions = %+v, want only yunobo memory", yunobo.Conclusions)
	}
}

func TestGormesMultiProfileSharedWorkspaceOptIn(t *testing.T) {
	ctx := context.Background()
	store, err := memory.OpenSqlite(t.TempDir()+"/memory.db", 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(ctx)
	if err := RunMigrations(store.DB()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	svc := NewService(store.DB(), Config{WorkspaceID: "gormes", ObserverPeerID: "gormes-runtime"}, nil)
	_, err = svc.Conclude(ctx, ConcludeParams{
		ProfileID:  "mineru",
		Peer:       "telegram:6586915095",
		Scope:      MemoryScopeWorkspace,
		Conclusion: "All profiles share the repo root policy.",
	})
	if err != nil {
		t.Fatalf("shared conclude: %v", err)
	}

	privateSearch, err := svc.Search(ctx, SearchParams{
		ProfileID: "yunobo",
		Peer:      "telegram:6586915095",
		Query:     "repo root policy",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("private search: %v", err)
	}
	if len(privateSearch.Results) != 0 {
		t.Fatalf("private profile search saw workspace memory without opt-in: %+v", privateSearch.Results)
	}

	sharedSearch, err := svc.Search(ctx, SearchParams{
		ProfileID: "yunobo",
		Peer:      "telegram:6586915095",
		Scope:     MemoryScopeWorkspace,
		Query:     "repo root policy",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("workspace search: %v", err)
	}
	if len(sharedSearch.Results) != 1 || sharedSearch.Results[0].Content != "All profiles share the repo root policy." {
		t.Fatalf("workspace results = %+v, want shared memory", sharedSearch.Results)
	}
}

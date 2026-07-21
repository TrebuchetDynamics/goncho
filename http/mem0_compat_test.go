package gonchohttp

import (
	"context"
	"net/http"
	"testing"

	"github.com/TrebuchetDynamics/goncho/memory"
	goncho "github.com/TrebuchetDynamics/goncho/service"
)

func TestMem0CompatibleHTTPMemoryLifecyclePreservesEvidence(t *testing.T) {
	ctx := context.Background()
	store, err := memory.OpenSqlite(t.TempDir()+"/memory.db", 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close(ctx)
	if err := goncho.RunMigrations(store.DB()); err != nil {
		t.Fatal(err)
	}
	handler := NewServiceHandler(goncho.NewService(store.DB(), goncho.Config{WorkspaceID: "mem0-http", ObserverPeerID: "agent"}, nil))

	added := requestJSON[goncho.MemoryItem](t, handler, http.MethodPost, "/v1/memories", map[string]any{
		"id": "memory-1", "user_id": "user-1", "profile_id": "profile-1", "workspace_id": "untrusted-other-workspace", "content": "Maya prefers SQLite for local memory.", "metadata": map[string]string{"topic": "database"},
	}, http.StatusCreated)
	if added.ID != "memory-1" || added.WorkspaceID != "mem0-http" || len(added.EvidenceIDs) == 0 {
		t.Fatalf("added = %+v, want stable ID, bound workspace, and evidence", added)
	}

	search := postJSON[goncho.MemorySearchResult](t, handler, "/v1/memories/search", map[string]any{
		"user_id": "user-1", "profile_id": "profile-1", "query": "SQLite", "metadata": map[string]string{"topic": "database"},
	}, http.StatusOK)
	if search.Count != 1 || search.Items[0].ID != "memory-1" {
		t.Fatalf("search = %+v, want memory-1", search)
	}

	updated := requestJSON[goncho.MemoryItem](t, handler, http.MethodPut, "/v1/memories/memory-1", map[string]any{
		"user_id": "user-1", "profile_id": "profile-1", "content": "Maya prefers SQLite with WAL for local memory.",
	}, http.StatusOK)
	if updated.Revision <= added.Revision {
		t.Fatalf("updated revision = %d, want > %d", updated.Revision, added.Revision)
	}

	got := getJSON[goncho.MemoryItem](t, handler, "/v1/memories/memory-1?user_id=user-1&profile_id=profile-1", http.StatusOK)
	if got.Content != updated.Content {
		t.Fatalf("get content = %q, want %q", got.Content, updated.Content)
	}
	getJSON[httpError](t, handler, "/v1/memories/memory-1/history?profile_id=profile-1", http.StatusBadRequest)
	history := getJSON[goncho.MemoryHistoryResult](t, handler, "/v1/memories/memory-1/history?user_id=user-1&profile_id=profile-1", http.StatusOK)
	if history.Count != 2 || history.Events[0].Action != "update" || history.Events[1].Action != "add" {
		t.Fatalf("history = %+v, want update/add evidence", history)
	}

	deleted := requestJSON[goncho.MemoryItem](t, handler, http.MethodDelete, "/v1/memories/memory-1?user_id=user-1&profile_id=profile-1", nil, http.StatusOK)
	if !deleted.Deleted {
		t.Fatalf("deleted = %+v, want tombstone", deleted)
	}
	getJSON[httpError](t, handler, "/v1/memories/memory-1?user_id=user-1&profile_id=profile-1", http.StatusNotFound)
}

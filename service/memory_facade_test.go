package goncho

import (
	"context"
	"errors"
	"testing"

	"github.com/TrebuchetDynamics/goncho/memory"
)

func TestMemoryFacadeAddSearchUpdateDeleteHistoryWithStableIDs(t *testing.T) {
	ctx := context.Background()
	store, err := memory.OpenSqlite(t.TempDir()+"/memory.db", 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(ctx)
	if err := RunMigrations(store.DB()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	svc := NewService(store.DB(), Config{WorkspaceID: "facade-workspace", ObserverPeerID: "agent-alpha"}, nil)
	facade := NewMemoryFacade(svc)

	first, err := facade.Add(ctx, MemoryAddParams{
		ID:        "locomo-memory-1",
		UserID:    "user-1",
		AgentID:   "agent-alpha",
		RunID:     "run-1",
		Content:   "Maya likes blue vault archive clues.",
		Metadata:  map[string]string{"topic": "vault", "source": "locomo"},
		ProfileID: "mineru",
	})
	if err != nil {
		t.Fatalf("Add first: %v", err)
	}
	second, err := facade.Add(ctx, MemoryAddParams{
		ID:        "locomo-memory-2",
		UserID:    "user-1",
		AgentID:   "agent-alpha",
		RunID:     "run-1",
		Content:   "Maya likes blue vault archive clues.",
		Metadata:  map[string]string{"topic": "duplicate", "source": "locomo"},
		ProfileID: "mineru",
	})
	if err != nil {
		t.Fatalf("Add second duplicate content: %v", err)
	}
	if first.ID != "locomo-memory-1" || second.ID != "locomo-memory-2" || first.ID == second.ID {
		t.Fatalf("stable IDs = %q/%q, want caller-supplied duplicate-safe IDs", first.ID, second.ID)
	}

	search, err := facade.Search(ctx, MemorySearchParams{UserID: "user-1", ProfileID: "mineru", Query: "blue vault", Metadata: map[string]string{"topic": "vault"}, Limit: 10})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(search.Items) != 1 || search.Items[0].ID != "locomo-memory-1" || search.Items[0].EvidenceIDs[0] == "" {
		t.Fatalf("search items = %+v, want first stable ID with evidence", search.Items)
	}

	updated, err := facade.Update(ctx, MemoryUpdateParams{ID: "locomo-memory-1", UserID: "user-1", ProfileID: "mineru", Content: "Maya likes green vault archive clues.", Metadata: map[string]string{"topic": "vault", "source": "corrected"}})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Content != "Maya likes green vault archive clues." || updated.Revision <= first.Revision {
		t.Fatalf("updated = %+v, want new content and higher revision", updated)
	}

	deleted, err := facade.Delete(ctx, MemoryDeleteParams{ID: "locomo-memory-1", UserID: "user-1", ProfileID: "mineru"})
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !deleted.Deleted {
		t.Fatalf("deleted = %+v, want tombstone", deleted)
	}
	postDelete, err := facade.Search(ctx, MemorySearchParams{UserID: "user-1", ProfileID: "mineru", Query: "green vault", Limit: 10})
	if err != nil {
		t.Fatalf("Search after delete: %v", err)
	}
	if len(postDelete.Items) != 0 {
		t.Fatalf("post-delete search = %+v, want deleted memory hidden", postDelete.Items)
	}

	history, err := facade.History(ctx, MemoryHistoryParams{ID: "locomo-memory-1", UserID: "user-1", ProfileID: "mineru", Limit: 10})
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if got := historyActions(history.Events); len(got) != 3 || got[0] != "delete" || got[1] != "update" || got[2] != "add" {
		t.Fatalf("history actions = %v, want delete/update/add newest first", got)
	}
	if !historyContainsContent(history.Events, "Maya likes blue vault archive clues.") || !historyContainsContent(history.Events, "Maya likes green vault archive clues.") {
		t.Fatalf("history events = %+v, want old and new content evidence", history.Events)
	}

	_, err = facade.Get(ctx, MemoryGetParams{ID: "locomo-memory-1", UserID: "user-1", ProfileID: "mineru"})
	if !errors.Is(err, ErrMemoryNotFound) {
		t.Fatalf("Get deleted err = %v, want ErrMemoryNotFound", err)
	}
}

func historyActions(events []MemoryHistoryEvent) []string {
	out := make([]string, 0, len(events))
	for _, event := range events {
		out = append(out, event.Action)
	}
	return out
}

func historyContainsContent(events []MemoryHistoryEvent, want string) bool {
	for _, event := range events {
		if event.PreviousContent == want || event.NewContent == want {
			return true
		}
	}
	return false
}

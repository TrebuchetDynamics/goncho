package goncho

import (
	"context"
	"errors"
	"testing"

	"github.com/TrebuchetDynamics/goncho/memory"
	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
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

func TestMemoryFacadeEvidenceIDsDisambiguateSameCallerIDAcrossUsers(t *testing.T) {
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

	left, err := facade.Add(ctx, MemoryAddParams{ID: "shared-memory-id", UserID: "user-left", ProfileID: "mineru", Content: "Left user likes blue archive clues."})
	if err != nil {
		t.Fatalf("Add left: %v", err)
	}
	right, err := facade.Add(ctx, MemoryAddParams{ID: "shared-memory-id", UserID: "user-right", ProfileID: "mineru", Content: "Right user likes green archive clues."})
	if err != nil {
		t.Fatalf("Add right: %v", err)
	}
	leftSlotEvidence := left.EvidenceIDs[len(left.EvidenceIDs)-1]
	rightSlotEvidence := right.EvidenceIDs[len(right.EvidenceIDs)-1]
	if leftSlotEvidence == rightSlotEvidence {
		t.Fatalf("slot evidence IDs collided: left=%q right=%q; want scoped provenance for same caller memory id", leftSlotEvidence, rightSlotEvidence)
	}
}

func TestMemoryFacadeSearchRejectsPunctuationOnlyQuery(t *testing.T) {
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

	if _, err := facade.Add(ctx, MemoryAddParams{ID: "punctuation-target", UserID: "user-1", ProfileID: "mineru", Content: "Maya likes blue archive clues."}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := facade.Search(ctx, MemorySearchParams{UserID: "user-1", ProfileID: "mineru", Query: "?!...", Limit: 10})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got.Items) != 0 {
		t.Fatalf("search items = %+v, want punctuation-only non-empty query not to match all memories", got.Items)
	}
}

func TestMemoryFacadeSearchReturnsNewestMatchingSlotsBeforeApplyingLimit(t *testing.T) {
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

	if _, err := facade.Add(ctx, MemoryAddParams{ID: "a-old", UserID: "user-1", ProfileID: "mineru", Content: "Maya likes blue archive clues."}); err != nil {
		t.Fatalf("Add old: %v", err)
	}
	if _, err := facade.Add(ctx, MemoryAddParams{ID: "z-new", UserID: "user-1", ProfileID: "mineru", Content: "Maya likes blue archive clues."}); err != nil {
		t.Fatalf("Add new: %v", err)
	}
	if _, err := svc.db.ExecContext(ctx, `UPDATE goncho_memory_slots SET updated_at = CASE name WHEN 'a-old' THEN 10 WHEN 'z-new' THEN 20 ELSE updated_at END WHERE peer_id = ?`, "user-1"); err != nil {
		t.Fatalf("force updated_at ordering: %v", err)
	}

	got, err := facade.Search(ctx, MemorySearchParams{UserID: "user-1", ProfileID: "mineru", Query: "blue archive", Limit: 1})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got.Items) != 1 || got.Items[0].ID != "z-new" {
		t.Fatalf("search items = %+v, want newest matching slot before limit", got.Items)
	}
}

func TestMemoryFacadeSearchScansPastInitialNonMatchingSlots(t *testing.T) {
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

	for _, id := range []string{"a-decoy-1", "a-decoy-2", "a-decoy-3", "a-decoy-4"} {
		if _, err := facade.Add(ctx, MemoryAddParams{ID: id, UserID: "user-1", ProfileID: "mineru", Content: "Maya likes blue archive clues.", Metadata: map[string]string{"topic": "decoy"}}); err != nil {
			t.Fatalf("Add decoy %s: %v", id, err)
		}
	}
	if _, err := facade.Add(ctx, MemoryAddParams{ID: "z-target", UserID: "user-1", ProfileID: "mineru", Content: "Maya likes blue archive clues.", Metadata: map[string]string{"topic": "vault"}}); err != nil {
		t.Fatalf("Add target: %v", err)
	}

	got, err := facade.Search(ctx, MemorySearchParams{UserID: "user-1", ProfileID: "mineru", Query: "blue archive", Metadata: map[string]string{"topic": "vault"}, Limit: 1})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got.Items) != 1 || got.Items[0].ID != "z-target" {
		t.Fatalf("search items = %+v, want target after initial nonmatching slots", got.Items)
	}
}

func TestMemoryFacadeHistoryScansPastInitialNonMatchingObservations(t *testing.T) {
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
	if _, err := facade.Add(ctx, MemoryAddParams{ID: "target", UserID: "user-1", ProfileID: "mineru", Content: "Maya likes blue vault archive clues."}); err != nil {
		t.Fatalf("Add target: %v", err)
	}
	for _, id := range []string{"newer-1", "newer-2", "newer-3", "newer-4"} {
		if _, err := facade.Add(ctx, MemoryAddParams{ID: id, UserID: "user-1", ProfileID: "mineru", Content: "Maya likes green archive clues."}); err != nil {
			t.Fatalf("Add newer %s: %v", id, err)
		}
	}

	got, err := facade.History(ctx, MemoryHistoryParams{ID: "target", UserID: "user-1", ProfileID: "mineru", Limit: 1})
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(got.Events) != 1 || got.Events[0].MemoryID != "target" || got.Events[0].Action != "add" {
		t.Fatalf("history events = %+v, want target event after initial nonmatching observations", got.Events)
	}
}

func historyActions(events []MemoryHistoryEvent) []string {
	return sliceutil.Map(events, func(event MemoryHistoryEvent) string { return event.Action })
}

func historyContainsContent(events []MemoryHistoryEvent, want string) bool {
	return sliceutil.ContainsFunc(events, func(event MemoryHistoryEvent) bool {
		return event.PreviousContent == want || event.NewContent == want
	})
}

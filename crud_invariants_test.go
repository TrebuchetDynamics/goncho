package goncho

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func TestCRUDInvariants_CreateMessagesAutocreatesSessionPeerAndSequences(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	createdAt := time.Unix(1700000100, 0).UTC()
	first, err := svc.CreateMessages(ctx, CreateMessagesParams{
		SessionKey: "sess-crud",
		Messages: []CreateMessage{
			{Peer: "alice", Content: "first", Metadata: map[string]any{"kind": "question"}, CreatedAt: createdAt},
			{Peer: "bob", Content: "second", Role: "assistant", CreatedAt: createdAt.Add(time.Second)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.CreateMessages(ctx, CreateMessagesParams{
		SessionKey: "sess-crud",
		Messages:   []CreateMessage{{Peer: "alice", Content: "third", CreatedAt: createdAt.Add(2 * time.Second)}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if got, want := messageSequences(append(first.Messages, second.Messages...)), []int{1, 2, 3}; !equalInts(got, want) {
		t.Fatalf("sequences = %#v, want %#v", got, want)
	}
	if first.Messages[0].WorkspaceID != "default" || first.Messages[0].SessionKey != "sess-crud" || first.Messages[0].Peer != "alice" {
		t.Fatalf("first message identity = %+v", first.Messages[0])
	}
	if first.Messages[0].Metadata["kind"] != "question" {
		t.Fatalf("metadata = %+v, want kind=question", first.Messages[0].Metadata)
	}

	rows := readLifecycleMessages(t, svc.db, "default", "sess-crud")
	if got, want := messageSequences(rows), []int{1, 2, 3}; !equalInts(got, want) {
		t.Fatalf("persisted sequences = %#v, want %#v", got, want)
	}
	if rows[1].Peer != "bob" || rows[1].Role != "assistant" {
		t.Fatalf("second row = %+v, want bob assistant", rows[1])
	}
}

func TestCRUDInvariants_DeleteSessionCascadesSessionScopedRowsOnly(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	if err := svc.SetProfile(ctx, "alice", []string{"keeps cross-session profile"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateMessages(ctx, CreateMessagesParams{
		SessionKey: "sess-delete",
		Messages:   []CreateMessage{{Peer: "alice", Content: "session message"}},
	}); err != nil {
		t.Fatal(err)
	}
	sessionConclusion, err := svc.Conclude(ctx, ConcludeParams{Peer: "alice", Conclusion: "session fact", SessionKey: "sess-delete"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Conclude(ctx, ConcludeParams{Peer: "alice", Conclusion: "global fact"}); err != nil {
		t.Fatal(err)
	}
	if err := upsertSessionSummary(ctx, svc.db, sessionSummaryRow{
		WorkspaceID: svc.workspaceID,
		SessionKey:  "sess-delete",
		SummaryType: "short",
		Content:     "summary",
		MessageID:   1,
		CreatedAt:   time.Now().Unix(),
		TokenCount:  2,
	}); err != nil {
		t.Fatal(err)
	}

	deleted, err := svc.DeleteSession(ctx, "sess-delete")
	if err != nil {
		t.Fatal(err)
	}
	if deleted.MessagesDeleted != 1 || deleted.ConclusionsDeleted != 1 || deleted.SummariesDeleted != 1 {
		t.Fatalf("delete counts = %+v, want one message/conclusion/summary", deleted)
	}
	if countLifecycleMessages(t, svc.db, "default", "sess-delete") != 0 {
		t.Fatal("session-scoped messages survived DeleteSession")
	}
	if countRows(t, svc.db, `SELECT COUNT(*) FROM goncho_conclusions WHERE id = ?`, sessionConclusion.ID) != 0 {
		t.Fatal("session-scoped conclusion survived DeleteSession")
	}
	if countRows(t, svc.db, `SELECT COUNT(*) FROM goncho_conclusions WHERE workspace_id = ? AND peer_id = ?`, "default", "alice") != 1 {
		t.Fatal("global peer conclusion should survive DeleteSession")
	}
	profile, err := svc.Profile(ctx, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if len(profile.Card) != 1 {
		t.Fatalf("peer card = %+v, want preserved cross-session profile", profile.Card)
	}
}

func TestCRUDInvariants_DeleteWorkspaceCascadesAndPreservesOtherWorkspaces(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	svc.dreamEnabled = true
	other := NewService(svc.db, Config{WorkspaceID: "other", ObserverPeerID: "gormes", DreamEnabled: true}, nil)
	svc.dreamEnabled = true
	other.dreamEnabled = true

	ctx := context.Background()
	for _, service := range []*Service{svc, other} {
		if err := service.SetProfile(ctx, "alice", []string{"profile"}); err != nil {
			t.Fatal(err)
		}
		if _, err := service.CreateMessages(ctx, CreateMessagesParams{
			SessionKey: "shared-session",
			Messages:   []CreateMessage{{Peer: "alice", Content: service.workspaceID + " message"}},
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := service.Conclude(ctx, ConcludeParams{Peer: "alice", Conclusion: service.workspaceID + " fact", SessionKey: "shared-session"}); err != nil {
			t.Fatal(err)
		}
		if err := upsertSessionSummary(ctx, service.db, sessionSummaryRow{
			WorkspaceID: service.workspaceID,
			SessionKey:  "shared-session",
			SummaryType: "short",
			Content:     "summary " + service.workspaceID,
			MessageID:   1,
			CreatedAt:   time.Now().Unix(),
			TokenCount:  2,
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := service.ScheduleDream(ctx, DreamScheduleParams{
			Peer:   "alice",
			Manual: true,
			Reason: "crud workspace delete fixture",
			Now:    time.Now(),
		}); err != nil {
			t.Fatal(err)
		}
	}

	deleted, err := svc.DeleteWorkspace(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if deleted.MessagesDeleted != 1 || deleted.PeerCardsDeleted != 1 || deleted.ConclusionsDeleted != 1 || deleted.SummariesDeleted != 1 || deleted.DreamsDeleted != 1 {
		t.Fatalf("workspace delete counts = %+v, want one row for each local Goncho table", deleted)
	}
	if countLifecycleMessages(t, svc.db, "default", "shared-session") != 0 {
		t.Fatal("default workspace messages survived DeleteWorkspace")
	}
	if countRows(t, svc.db, `SELECT COUNT(*) FROM goncho_peer_cards WHERE workspace_id = ?`, "default") != 0 {
		t.Fatal("default peer cards survived DeleteWorkspace")
	}
	if countLifecycleMessages(t, svc.db, "other", "shared-session") != 1 {
		t.Fatal("other workspace messages should survive DeleteWorkspace")
	}
	if countRows(t, svc.db, `SELECT COUNT(*) FROM goncho_peer_cards WHERE workspace_id = ?`, "other") != 1 {
		t.Fatal("other workspace peer cards should survive DeleteWorkspace")
	}
	if countRows(t, svc.db, `SELECT COUNT(*) FROM goncho_dreams WHERE workspace_id = ?`, "other") != 1 {
		t.Fatal("other workspace dream rows should survive DeleteWorkspace")
	}
}

func readLifecycleMessages(t *testing.T, db *sql.DB, workspaceID, sessionKey string) []MessageRecord {
	t.Helper()
	got, err := listLifecycleMessages(context.Background(), db, workspaceID, sessionKey)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func countLifecycleMessages(t *testing.T, db *sql.DB, workspaceID, sessionKey string) int {
	t.Helper()
	return len(readLifecycleMessages(t, db, workspaceID, sessionKey))
}

func countRows(t *testing.T, db *sql.DB, query string, args ...any) int {
	t.Helper()
	var count int
	if err := db.QueryRow(query, args...).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func messageSequences(messages []MessageRecord) []int {
	out := make([]int, 0, len(messages))
	for _, msg := range messages {
		out = append(out, msg.Sequence)
	}
	return out
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

package goncho

import (
	"context"
	"encoding/json"
	"slices"
	"testing"
	"time"

	memory "github.com/TrebuchetDynamics/goncho/memory"
	session "github.com/TrebuchetDynamics/goncho/session"
)

func TestService_ProfileRoundTrip(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	if err := svc.SetProfile(ctx, "telegram:6586915095", []string{"Blind", "Prefers exact outputs"}); err != nil {
		t.Fatal(err)
	}

	got, err := svc.Profile(ctx, "telegram:6586915095")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Blind", "Prefers exact outputs"}
	if !slices.Equal(got.Card, want) {
		t.Fatalf("card = %#v, want %#v", got.Card, want)
	}
}

func TestService_ConcludeAndSearchIsIdempotent(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	first, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "telegram:6586915095",
		Conclusion: "The user prefers exact evidence-first reports.",
		SessionKey: "telegram:6586915095",
	})
	if err != nil {
		t.Fatal(err)
	}

	second, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "telegram:6586915095",
		Conclusion: "The user prefers exact evidence-first reports.",
		SessionKey: "telegram:6586915095",
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatalf("idempotent conclude returned ids %d and %d", first.ID, second.ID)
	}

	got, err := svc.Search(ctx, SearchParams{
		Peer:       "telegram:6586915095",
		Query:      "evidence-first",
		MaxTokens:  200,
		SessionKey: "telegram:6586915095",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Results) == 0 {
		t.Fatal("want at least one search result")
	}
	if got.Results[0].Source != "conclusion" {
		t.Fatalf("first result source = %q, want conclusion", got.Results[0].Source)
	}
}

func TestService_ContextIncludesPeerCardConclusionsAndRecentMessages(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	if err := svc.SetProfile(ctx, "telegram:6586915095", []string{"Blind", "Prefers exact outputs"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "telegram:6586915095",
		Conclusion: "The user prefers exact evidence-first reports.",
		SessionKey: "telegram:6586915095",
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.db.ExecContext(ctx,
		`INSERT INTO turns(session_id, role, content, ts_unix, chat_id) VALUES (?, ?, ?, ?, ?)`,
		"sess-1", "user", "Please keep reports exact and evidence-first.", time.Now().Unix(), "telegram:6586915095",
	); err != nil {
		t.Fatal(err)
	}

	got, err := svc.Context(ctx, ContextParams{
		Peer:       "telegram:6586915095",
		Query:      "exact",
		MaxTokens:  400,
		SessionKey: "telegram:6586915095",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.PeerCard) != 2 {
		t.Fatalf("peer card len = %d, want 2", len(got.PeerCard))
	}
	if len(got.Conclusions) == 0 {
		t.Fatal("want conclusions in context")
	}
	if len(got.RecentMessages) == 0 {
		t.Fatal("want recent messages in context")
	}
	if got.Representation == "" {
		t.Fatal("want non-empty representation")
	}
}

func TestService_SkipsInterruptedTurnsInSearchAndContext(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().Unix()
	if _, err := svc.db.ExecContext(ctx,
		`INSERT INTO turns(session_id, role, content, ts_unix, chat_id, memory_sync_status, memory_sync_reason)
		 VALUES
		 ('sess-ready', 'user', 'stable mango preference', ?, 'telegram:6586915095', 'ready', NULL),
		 ('sess-skip', 'user', 'interrupted pineapple draft', ?, 'telegram:6586915095', 'skipped', 'interrupted')`,
		now, now+1,
	); err != nil {
		t.Fatal(err)
	}

	search, err := svc.Search(ctx, SearchParams{
		Peer:       "telegram:6586915095",
		Query:      "pineapple",
		MaxTokens:  200,
		SessionKey: "telegram:6586915095",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(search.Results) != 0 {
		t.Fatalf("Search returned skipped turn results: %+v", search.Results)
	}

	got, err := svc.Context(ctx, ContextParams{
		Peer:       "telegram:6586915095",
		MaxTokens:  400,
		SessionKey: "telegram:6586915095",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.RecentMessages) != 1 || got.RecentMessages[0].Content != "stable mango preference" {
		t.Fatalf("RecentMessages = %+v, want only ready turn", got.RecentMessages)
	}
}

func TestService_DeleteConclusion(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	created, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "telegram:6586915095",
		Conclusion: "The user dislikes vague status reports.",
	})
	if err != nil {
		t.Fatal(err)
	}

	deleted, err := svc.Conclude(ctx, ConcludeParams{
		Peer:     "telegram:6586915095",
		DeleteID: created.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !deleted.Deleted {
		t.Fatal("expected deleted=true")
	}

	got, err := svc.Search(ctx, SearchParams{
		Peer:      "telegram:6586915095",
		Query:     "vague",
		MaxTokens: 100,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Results) != 0 {
		t.Fatalf("expected 0 search results after delete, got %d", len(got.Results))
	}
}

func TestService_SearchUserScopeRespectsSourceFilter(t *testing.T) {
	store, dir, svc, cleanup := newTestServiceWithDirectory(t)
	defer cleanup()

	ctx := context.Background()
	for _, meta := range []session.Metadata{
		{SessionID: "sess-telegram", Source: "telegram", ChatID: "42", UserID: "user-juan"},
		{SessionID: "sess-discord", Source: "discord", ChatID: "chan-9", UserID: "user-juan"},
	} {
		if err := dir.PutMetadata(ctx, meta); err != nil {
			t.Fatalf("PutMetadata(%s): %v", meta.SessionID, err)
		}
	}
	now := time.Now().Unix()
	for _, turn := range []struct {
		sessionID string
		chatID    string
		content   string
		ts        int64
	}{
		{
			sessionID: "sess-telegram",
			chatID:    "telegram:42",
			content:   "Atlas planning stayed in Telegram.",
			ts:        now - 20,
		},
		{
			sessionID: "sess-discord",
			chatID:    "discord:chan-9",
			content:   "Atlas execution moved to Discord.",
			ts:        now - 10,
		},
	} {
		if _, err := store.DB().ExecContext(ctx,
			`INSERT INTO turns(session_id, role, content, ts_unix, chat_id) VALUES (?, ?, ?, ?, ?)`,
			turn.sessionID, "user", turn.content, turn.ts, turn.chatID,
		); err != nil {
			t.Fatalf("insert turn %s: %v", turn.sessionID, err)
		}
	}

	got, err := svc.Search(ctx, SearchParams{
		Peer:       "user-juan",
		Query:      "Atlas",
		MaxTokens:  200,
		SessionKey: "discord:chan-9",
		Scope:      "user",
		Sources:    []string{"discord"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Results) != 1 {
		t.Fatalf("Search results len = %d, want 1", len(got.Results))
	}
	if got.Results[0].Source != "turn" || got.Results[0].SessionKey != "sess-discord" {
		t.Fatalf("Search result = %+v, want discord turn bound to sess-discord", got.Results[0])
	}
	if got.Results[0].OriginSource != "discord" {
		t.Fatalf("Search result origin_source = %q, want discord source allowlist evidence", got.Results[0].OriginSource)
	}
}

func TestServiceSearchUserScopeReturnsLineageEvidenceForWidenedHits(t *testing.T) {
	store, dir, svc, cleanup := newTestServiceWithDirectory(t)
	defer cleanup()

	ctx := context.Background()
	for _, meta := range []session.Metadata{
		{SessionID: "sess-current", Source: "discord", ChatID: "chan-9", UserID: "user-juan"},
		{SessionID: "sess-parent", Source: "telegram", ChatID: "42", UserID: "user-juan"},
		{
			SessionID:       "sess-child",
			Source:          "telegram",
			ChatID:          "42",
			UserID:          "user-juan",
			ParentSessionID: "sess-parent",
			LineageKind:     session.LineageKindCompression,
		},
		{
			SessionID:       "sess-orphan",
			Source:          "telegram",
			ChatID:          "42",
			UserID:          "user-juan",
			ParentSessionID: "sess-missing",
			LineageKind:     session.LineageKindFork,
		},
	} {
		if err := dir.PutMetadata(ctx, meta); err != nil {
			t.Fatalf("PutMetadata(%s): %v", meta.SessionID, err)
		}
	}
	now := time.Now().Unix()
	for _, turn := range []struct {
		sessionID string
		chatID    string
		content   string
		ts        int64
	}{
		{"sess-current", "discord:chan-9", "Atlas current Discord fallback evidence.", now - 40},
		{"sess-child", "telegram:42", "Atlas child Telegram lineage evidence.", now - 30},
		{"sess-orphan", "telegram:42", "Atlas orphan Telegram lineage evidence.", now - 20},
		{"sess-chat-only", "telegram:42", "Atlas legacy chat-only lineage evidence.", now - 10},
	} {
		if _, err := store.DB().ExecContext(ctx,
			`INSERT INTO turns(session_id, role, content, ts_unix, chat_id) VALUES (?, 'user', ?, ?, ?)`,
			turn.sessionID, turn.content, turn.ts, turn.chatID,
		); err != nil {
			t.Fatalf("insert turn %s: %v", turn.sessionID, err)
		}
	}

	got, err := svc.Search(ctx, SearchParams{
		Peer:       "user-juan",
		Query:      "Atlas",
		MaxTokens:  400,
		SessionKey: "discord:chan-9",
		Scope:      "user",
		Sources:    []string{"telegram"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ScopeEvidence == nil ||
		got.ScopeEvidence.Decision != memory.CrossChatDecisionAllowed ||
		got.ScopeEvidence.SessionsConsidered != 3 ||
		got.ScopeEvidence.WidenedSessionsConsidered != 3 ||
		!slices.Equal(got.ScopeEvidence.SourceAllowlist, []string{"telegram"}) {
		t.Fatalf("ScopeEvidence = %+v, want allowed telegram lineage evidence", got.ScopeEvidence)
	}

	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("Marshal SearchResultSet: %v", err)
	}
	var wire serviceSearchEvidenceWire
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatalf("Unmarshal SearchResultSet wire shape: %v\n%s", err, raw)
	}
	if len(wire.Results) != 3 {
		t.Fatalf("Results len = %d, want 3: %+v", len(wire.Results), wire.Results)
	}
	hits := make(map[string]serviceSearchHitWire, len(wire.Results))
	for _, hit := range wire.Results {
		hits[hit.SessionKey] = hit
	}
	assertServiceSearchLineageEvidence(t, hits, "sess-child", "telegram", "ok", "sess-parent", "compression")
	assertServiceSearchLineageEvidence(t, hits, "sess-orphan", "telegram", "orphan", "sess-missing", "fork")
	assertServiceSearchLineageEvidence(t, hits, "sess-chat-only", "telegram", "unavailable", "", "")
}

func TestService_SearchUserScopeUnknownCurrentBindingFallsBackSameSession(t *testing.T) {
	store, dir, svc, cleanup := newTestServiceWithDirectory(t)
	defer cleanup()

	ctx := context.Background()
	if err := dir.PutMetadata(ctx, session.Metadata{
		SessionID: "sess-telegram",
		Source:    "telegram",
		ChatID:    "42",
		UserID:    "user-juan",
	}); err != nil {
		t.Fatalf("PutMetadata telegram: %v", err)
	}
	now := time.Now().Unix()
	for _, turn := range []struct {
		sessionID string
		chatID    string
		content   string
		ts        int64
	}{
		{
			sessionID: "sess-telegram",
			chatID:    "telegram:42",
			content:   "Atlas remote user-scope note.",
			ts:        now - 20,
		},
		{
			sessionID: "sess-current",
			chatID:    "discord:chan-9",
			content:   "Atlas same-session fallback note.",
			ts:        now - 10,
		},
	} {
		if _, err := store.DB().ExecContext(ctx,
			`INSERT INTO turns(session_id, role, content, ts_unix, chat_id) VALUES (?, ?, ?, ?, ?)`,
			turn.sessionID, "user", turn.content, turn.ts, turn.chatID,
		); err != nil {
			t.Fatalf("insert turn %s: %v", turn.sessionID, err)
		}
	}

	got, err := svc.Search(ctx, SearchParams{
		Peer:       "user-juan",
		Query:      "Atlas",
		MaxTokens:  200,
		SessionKey: "discord:chan-9",
		Scope:      "user",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Results) != 1 {
		t.Fatalf("Search results len = %d, want 1", len(got.Results))
	}
	if got.Results[0].Content != "Atlas same-session fallback note." {
		t.Fatalf("Search result = %+v, want same-session fallback only", got.Results[0])
	}
}

type serviceSearchEvidenceWire struct {
	Results []serviceSearchHitWire `json:"results"`
}

type serviceSearchHitWire struct {
	Source       string                   `json:"source"`
	OriginSource string                   `json:"origin_source"`
	SessionKey   string                   `json:"session_key"`
	Lineage      serviceSearchLineageWire `json:"lineage"`
}

type serviceSearchLineageWire struct {
	Status          string `json:"status"`
	ParentSessionID string `json:"parent_session_id"`
	LineageKind     string `json:"lineage_kind"`
}

func assertServiceSearchLineageEvidence(t *testing.T, hits map[string]serviceSearchHitWire, sessionKey, originSource, status, parentSessionID, lineageKind string) {
	t.Helper()

	hit, ok := hits[sessionKey]
	if !ok {
		t.Fatalf("missing search hit for %s in %+v", sessionKey, hits)
	}
	if hit.Source != "turn" || hit.OriginSource != originSource {
		t.Fatalf("hit %s = %+v, want turn from %s", sessionKey, hit, originSource)
	}
	if hit.Lineage.Status != status ||
		hit.Lineage.ParentSessionID != parentSessionID ||
		hit.Lineage.LineageKind != lineageKind {
		t.Fatalf("hit %s lineage = %+v, want status %q parent %q kind %q",
			sessionKey, hit.Lineage, status, parentSessionID, lineageKind)
	}
}

func newTestService(t *testing.T) (*Service, func()) {
	t.Helper()

	store, err := memory.OpenSqlite(t.TempDir()+"/memory.db", 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}

	svc := NewService(store.DB(), Config{
		WorkspaceID:    "default",
		ObserverPeerID: "gormes",
		RecentMessages: 4,
	}, nil)

	return svc, func() {
		if err := store.Close(context.Background()); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}
}

func newTestServiceWithDirectory(t *testing.T) (*memory.SqliteStore, *session.MemMap, *Service, func()) {
	t.Helper()

	store, err := memory.OpenSqlite(t.TempDir()+"/memory.db", 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	dir := session.NewMemMap()
	svc := NewService(store.DB(), Config{
		WorkspaceID:      "default",
		ObserverPeerID:   "gormes",
		RecentMessages:   4,
		SessionDirectory: &memMapAdapter{dir},
	}, nil)
	return store, dir, svc, func() {
		if err := store.Close(context.Background()); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}
}

type memMapAdapter struct {
	*session.MemMap
}

func (a *memMapAdapter) ListMetadataByUserID(ctx context.Context, userID string) ([]SessionMetadata, error) {
	metas, err := a.MemMap.ListMetadataByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]SessionMetadata, len(metas))
	for i, m := range metas {
		out[i] = SessionMetadata{
			SessionID:       m.SessionID,
			Source:          m.Source,
			ChatID:          m.ChatID,
			UserID:          m.UserID,
			Title:           m.Title,
			ParentSessionID: m.ParentSessionID,
			LineageKind:     m.LineageKind,
			CreatedAt:       m.CreatedAt,
			UpdatedAt:       m.UpdatedAt,
		}
	}
	return out, nil
}

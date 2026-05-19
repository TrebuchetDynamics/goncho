package goncho

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

)

func TestServiceSearchUserScopeReturnsOperatorEvidenceForAllowedAndDeniedRecall(t *testing.T) {
	store, dir, svc, cleanup := newTestServiceWithDirectory(t)
	defer cleanup()

	ctx := context.Background()
	for _, meta := range []SessionMetadata{
		{SessionID: "sess-current", Source: "discord", ChatID: "chan-9", UserID: "user-juan"},
		{SessionID: "sess-telegram", Source: "telegram", ChatID: "42", UserID: "user-juan"},
	} {
		if err := dir.PutMetadata(ctx, meta); err != nil {
			t.Fatalf("PutMetadata(%s): %v", meta.SessionID, err)
		}
	}
	now := time.Now().Unix()
	_, err := store.DB().ExecContext(ctx,
		`INSERT INTO turns(session_id, role, content, ts_unix, chat_id)
		 VALUES
		 ('sess-current', 'user', 'Atlas current Discord note.', ?, 'discord:chan-9'),
		 ('sess-telegram', 'user', 'Atlas widened Telegram note.', ?, 'telegram:42'),
		 ('sess-fallback', 'user', 'Atlas same-chat fallback note.', ?, 'slack:C123')`,
		now-30, now-20, now-10,
	)
	if err != nil {
		t.Fatal(err)
	}

	allowed, err := svc.Search(ctx, SearchParams{
		Peer:       "user-juan",
		Query:      "Atlas",
		SessionKey: "discord:chan-9",
		Scope:      "user",
		Sources:    []string{"telegram", "discord"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if allowed.ScopeEvidence == nil {
		t.Fatalf("ScopeEvidence = nil, want allowed operator evidence")
	}
	if allowed.ScopeEvidence.Decision != CrossChatDecisionAllowed {
		t.Fatalf("Decision = %q, want allowed: %+v", allowed.ScopeEvidence.Decision, allowed.ScopeEvidence)
	}
	if allowed.ScopeEvidence.UserID != "user-juan" || allowed.ScopeEvidence.WidenedSessionsConsidered != 1 {
		t.Fatalf("ScopeEvidence = %+v, want user-juan with one widened session", allowed.ScopeEvidence)
	}
	if !slices.Equal(allowed.ScopeEvidence.SourceAllowlist, []string{"telegram", "discord"}) {
		t.Fatalf("SourceAllowlist = %v, want request-order allowlist", allowed.ScopeEvidence.SourceAllowlist)
	}
	if len(allowed.Results) != 2 {
		t.Fatalf("Results len = %d, want 2: %+v", len(allowed.Results), allowed.Results)
	}
	if allowed.Results[0].OriginSource == "" || allowed.Results[1].OriginSource == "" {
		t.Fatalf("Results = %+v, want origin_source evidence on returned hits", allowed.Results)
	}

	denied, err := svc.Search(ctx, SearchParams{
		Peer:       "user-juan",
		Query:      "Atlas",
		SessionKey: "slack:C123",
		Scope:      "user",
	})
	if err != nil {
		t.Fatal(err)
	}
	if denied.ScopeEvidence == nil {
		t.Fatalf("ScopeEvidence = nil, want denied operator evidence")
	}
	if denied.ScopeEvidence.Decision != CrossChatDecisionDenied {
		t.Fatalf("Decision = %q, want denied: %+v", denied.ScopeEvidence.Decision, denied.ScopeEvidence)
	}
	if denied.ScopeEvidence.FallbackScope != CrossChatFallbackSameChat {
		t.Fatalf("FallbackScope = %q, want same-chat", denied.ScopeEvidence.FallbackScope)
	}
	if len(denied.Results) != 1 || denied.Results[0].Content != "Atlas same-chat fallback note." {
		t.Fatalf("Denied results = %+v, want same-chat fallback hit only", denied.Results)
	}
	if denied.Results[0].OriginSource != "slack" {
		t.Fatalf("Denied fallback origin_source = %q, want slack", denied.Results[0].OriginSource)
	}
}

func TestServiceContextUserScopeDryRunPreservesSearchHitOriginSourceEvidence(t *testing.T) {
	store, dir, svc, cleanup := newTestServiceWithDirectory(t)
	defer cleanup()

	ctx := context.Background()
	for _, meta := range []SessionMetadata{
		{SessionID: "sess-current", Source: "discord", ChatID: "chan-9", UserID: "user-juan"},
		{SessionID: "sess-telegram", Source: "telegram", ChatID: "42", UserID: "user-juan"},
	} {
		if err := dir.PutMetadata(ctx, meta); err != nil {
			t.Fatalf("PutMetadata(%s): %v", meta.SessionID, err)
		}
	}
	now := time.Now().Unix()
	_, err := store.DB().ExecContext(ctx,
		`INSERT INTO turns(session_id, role, content, ts_unix, chat_id)
		 VALUES
		 ('sess-current', 'user', 'doctor dry-run current Discord note.', ?, 'discord:chan-9'),
		 ('sess-telegram', 'user', 'doctor dry-run widened Telegram note.', ?, 'telegram:42')`,
		now-20, now-10,
	)
	if err != nil {
		t.Fatal(err)
	}

	got, err := svc.Context(ctx, ContextParams{
		Peer:       "user-juan",
		Query:      "doctor dry-run",
		SessionKey: "discord:chan-9",
		Scope:      "user",
		Sources:    []string{"telegram"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if got.ScopeEvidence == nil || got.ScopeEvidence.Decision != CrossChatDecisionAllowed {
		t.Fatalf("ScopeEvidence = %+v, want allowed user-scope evidence", got.ScopeEvidence)
	}
	if len(got.SearchResults) != 1 {
		t.Fatalf("SearchResults len = %d, want 1: %+v", len(got.SearchResults), got.SearchResults)
	}
	if got.SearchResults[0].OriginSource != "telegram" || got.SearchResults[0].SessionKey != "sess-telegram" {
		t.Fatalf("SearchResults[0] = %+v, want telegram origin_source on widened hit", got.SearchResults[0])
	}
}

func TestServiceSearchUserScopeConflictingCurrentBindingReportsConflictEvidence(t *testing.T) {
	store, dir, svc, cleanup := newTestServiceWithDirectory(t)
	defer cleanup()

	ctx := context.Background()
	for _, meta := range []SessionMetadata{
		{SessionID: "sess-current", Source: "discord", ChatID: "chan-9", UserID: "user-maria"},
		{SessionID: "sess-telegram", Source: "telegram", ChatID: "42", UserID: "user-juan"},
	} {
		if err := dir.PutMetadata(ctx, meta); err != nil {
			t.Fatalf("PutMetadata(%s): %v", meta.SessionID, err)
		}
	}
	now := time.Now().Unix()
	_, err := store.DB().ExecContext(ctx,
		`INSERT INTO turns(session_id, role, content, ts_unix, chat_id)
		 VALUES
		 ('sess-current', 'user', 'Atlas conflicting same-chat fallback note.', ?, 'discord:chan-9'),
		 ('sess-telegram', 'user', 'Atlas widened Telegram note must not be used.', ?, 'telegram:42')`,
		now-20, now-10,
	)
	if err != nil {
		t.Fatal(err)
	}

	got, err := svc.Search(ctx, SearchParams{
		Peer:       "user-juan",
		Query:      "Atlas",
		SessionKey: "discord:chan-9",
		Scope:      "user",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ScopeEvidence == nil {
		t.Fatal("ScopeEvidence = nil, want denied conflict evidence")
	}
	if got.ScopeEvidence.Decision != CrossChatDecisionDenied {
		t.Fatalf("Decision = %q, want denied: %+v", got.ScopeEvidence.Decision, got.ScopeEvidence)
	}
	if got.ScopeEvidence.FallbackScope != CrossChatFallbackSameChat {
		t.Fatalf("FallbackScope = %q, want same-chat", got.ScopeEvidence.FallbackScope)
	}
	if got.ScopeEvidence.Reason == "" || !strings.Contains(got.ScopeEvidence.Reason, "conflicting current binding") {
		t.Fatalf("Reason = %q, want conflicting current binding", got.ScopeEvidence.Reason)
	}
	if len(got.Results) != 1 || got.Results[0].Content != "Atlas conflicting same-chat fallback note." {
		t.Fatalf("Results = %+v, want same-chat fallback only", got.Results)
	}
}

package goncho

import (
	"context"
	"errors"
	"testing"
	"time"

	session "github.com/TrebuchetDynamics/goncho/session"
)

func TestService_SearchUnsupportedMetadataFilterFailsClosed(t *testing.T) {
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
	if _, err := store.DB().ExecContext(ctx,
		`INSERT INTO turns(session_id, role, content, ts_unix, chat_id)
		 VALUES
		 ('sess-telegram', 'user', 'Atlas remote metadata leak candidate.', ?, 'telegram:42'),
		 ('sess-discord', 'user', 'Atlas current session note.', ?, 'discord:chan-9')`,
		now-20, now-10,
	); err != nil {
		t.Fatal(err)
	}

	_, err := svc.Search(ctx, SearchParams{
		Peer:       "user-juan",
		Query:      "Atlas",
		SessionKey: "discord:chan-9",
		Scope:      "user",
		Filters: map[string]any{
			"metadata": map[string]any{"priority": "high"},
		},
	})
	var unsupported *UnsupportedFilterError
	if !errors.As(err, &unsupported) {
		t.Fatalf("Search err = %T %[1]v, want UnsupportedFilterError", err)
	}
	if unsupported.Field != "metadata.priority" {
		t.Fatalf("UnsupportedFilterError.Field = %q, want metadata.priority", unsupported.Field)
	}
}

func TestService_SearchSupportedFiltersKeepUserScopeNarrow(t *testing.T) {
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
	if _, err := store.DB().ExecContext(ctx,
		`INSERT INTO turns(session_id, role, content, ts_unix, chat_id)
		 VALUES
		 ('sess-telegram', 'user', 'Atlas Telegram note.', ?, 'telegram:42'),
		 ('sess-discord', 'user', 'Atlas Discord note.', ?, 'discord:chan-9')`,
		now-20, now-10,
	); err != nil {
		t.Fatal(err)
	}

	got, err := svc.Search(ctx, SearchParams{
		Peer:       "user-juan",
		Query:      "Atlas",
		SessionKey: "discord:chan-9",
		Scope:      "user",
		Filters: map[string]any{
			"AND": []any{
				map[string]any{"session_id": "sess-discord"},
				map[string]any{"source": "discord"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Results) != 1 {
		t.Fatalf("Search results len = %d, want 1: %+v", len(got.Results), got.Results)
	}
	if got.Results[0].SessionKey != "sess-discord" || got.Results[0].OriginSource != "discord" {
		t.Fatalf("Search result = %+v, want discord session only", got.Results[0])
	}
}

func TestService_SearchSessionFilterCannotWidenSameChatRecall(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().Unix()
	if _, err := svc.db.ExecContext(ctx,
		`INSERT INTO turns(session_id, role, content, ts_unix, chat_id)
		 VALUES
		 ('sess-telegram', 'user', 'Atlas remote same-chat leak candidate.', ?, 'telegram:42'),
		 ('sess-current', 'user', 'Atlas current same-chat note.', ?, 'discord:chan-9')`,
		now-20, now-10,
	); err != nil {
		t.Fatal(err)
	}

	got, err := svc.Search(ctx, SearchParams{
		Peer:       "user-juan",
		Query:      "Atlas",
		SessionKey: "discord:chan-9",
		Filters: map[string]any{
			"session_id": "sess-telegram",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Results) != 0 {
		t.Fatalf("Search returned widened same-chat results: %+v", got.Results)
	}
}

func TestService_SearchSourceFilterCannotWidenSameChatRecall(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().Unix()
	if _, err := svc.db.ExecContext(ctx,
		`INSERT INTO turns(session_id, role, content, ts_unix, chat_id)
		 VALUES ('sess-current', 'user', 'Atlas current Discord note.', ?, 'discord:chan-9')`,
		now,
	); err != nil {
		t.Fatal(err)
	}

	got, err := svc.Search(ctx, SearchParams{
		Peer:       "user-juan",
		Query:      "Atlas",
		SessionKey: "discord:chan-9",
		Filters: map[string]any{
			"source": "telegram",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Results) != 0 {
		t.Fatalf("Search returned same-chat results that do not match source filter: %+v", got.Results)
	}
}

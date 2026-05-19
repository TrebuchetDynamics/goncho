package goncho

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	session "github.com/TrebuchetDynamics/goncho/session"
)

func TestSearchFilterGrammarParsesHonchoOperators(t *testing.T) {
	expr, err := parseSearchFilter(map[string]any{
		"AND": []any{
			map[string]any{"session_id": "sess-discord"},
			map[string]any{"OR": []any{
				map[string]any{"created_at": map[string]any{
					"gt":  "2024-01-01T00:00:00Z",
					"gte": "2024-01-02T00:00:00Z",
					"lt":  "2024-02-01T00:00:00Z",
					"lte": "2024-02-02T00:00:00Z",
					"ne":  "2024-01-03T00:00:00Z",
				}},
				map[string]any{"peer_id": map[string]any{"in": []any{"alice", "bob", "*"}}},
			}},
			map[string]any{"NOT": []any{
				map[string]any{"content": map[string]any{"contains": "draft"}},
				map[string]any{"content": map[string]any{"icontains": "SECRET"}},
			}},
			map[string]any{"metadata": map[string]any{
				"profile": map[string]any{"department": "engineering"},
				"score":   map[string]any{"gt": 0.8},
			}},
		},
	})
	if err != nil {
		t.Fatalf("parseSearchFilter: %v", err)
	}

	if expr.Kind != filterKindAnd {
		t.Fatalf("root kind = %v, want %v", expr.Kind, filterKindAnd)
	}
	requireParsedComparison(t, expr, "session_id", filterOpEQ)
	requireParsedComparison(t, expr, "created_at", filterOpGT)
	requireParsedComparison(t, expr, "created_at", filterOpGTE)
	requireParsedComparison(t, expr, "created_at", filterOpLT)
	requireParsedComparison(t, expr, "created_at", filterOpLTE)
	requireParsedComparison(t, expr, "created_at", filterOpNE)
	requireParsedComparison(t, expr, "peer_id", filterOpIn)
	requireParsedComparison(t, expr, "content", filterOpContains)
	requireParsedComparison(t, expr, "content", filterOpIContains)
	requireParsedComparison(t, expr, "metadata.profile.department", filterOpEQ)
	requireParsedComparison(t, expr, "metadata.score", filterOpGT)
	if !containsWildcard(expr) {
		t.Fatalf("parsed expression %#v does not preserve wildcard value", expr)
	}
}

func TestSearchFilterGrammarRejectsUnknownFieldsAndOperators(t *testing.T) {
	tests := []struct {
		name      string
		filter    map[string]any
		wantField string
		wantOp    string
	}{
		{
			name:      "unknown field",
			filter:    map[string]any{"workspace_slug": "prod"},
			wantField: "workspace_slug",
		},
		{
			name:      "unknown operator",
			filter:    map[string]any{"created_at": map[string]any{"regex": "2024"}},
			wantField: "created_at",
			wantOp:    "regex",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseSearchFilter(tt.filter)
			var unsupported *UnsupportedFilterError
			if !errors.As(err, &unsupported) {
				t.Fatalf("parseSearchFilter err = %T %[1]v, want UnsupportedFilterError", err)
			}
			if unsupported.Field != tt.wantField {
				t.Fatalf("UnsupportedFilterError.Field = %q, want %q", unsupported.Field, tt.wantField)
			}
			if unsupported.Operator != tt.wantOp {
				t.Fatalf("UnsupportedFilterError.Operator = %q, want %q", unsupported.Operator, tt.wantOp)
			}
			if unsupported.Code != "unsupported_filter" || unsupported.Reason == "" {
				t.Fatalf("UnsupportedFilterError = %+v, want structured unsupported-filter evidence", unsupported)
			}
		})
	}
}

func TestSearchFilterCompilerSupportsSessionSourcePeerAndRejectsMetadata(t *testing.T) {
	supported, err := compileSearchFilter(mustParseSearchFilter(t, map[string]any{
		"AND": []any{
			map[string]any{"session_id": map[string]any{"in": []any{"sess-discord", "*"}}},
			map[string]any{"source": "discord"},
			map[string]any{"peer_id": "user-juan"},
		},
	}), "user-juan")
	if err != nil {
		t.Fatalf("compileSearchFilter supported subset: %v", err)
	}
	if !slices.Equal(supported.SessionIDs, []string{"sess-discord", "*"}) {
		t.Fatalf("SessionIDs = %#v, want sess-discord and wildcard", supported.SessionIDs)
	}
	if !slices.Equal(supported.Sources, []string{"discord"}) {
		t.Fatalf("Sources = %#v, want discord", supported.Sources)
	}
	if supported.DenyAll {
		t.Fatal("DenyAll = true for matching peer_id")
	}

	unsupportedExpr := mustParseSearchFilter(t, map[string]any{
		"metadata": map[string]any{"priority": "high"},
	})
	_, err = compileSearchFilter(unsupportedExpr, "user-juan")
	var unsupported *UnsupportedFilterError
	if !errors.As(err, &unsupported) {
		t.Fatalf("compileSearchFilter metadata err = %T %[1]v, want UnsupportedFilterError", err)
	}
	if unsupported.Field != "metadata.priority" {
		t.Fatalf("UnsupportedFilterError.Field = %q, want metadata.priority", unsupported.Field)
	}
}

func TestSearchLimitDefaultsToTenAndClampsAtHonchoMaximum(t *testing.T) {
	tests := []struct {
		raw  int
		want int
	}{
		{raw: 0, want: 10},
		{raw: -5, want: 10},
		{raw: 7, want: 7},
		{raw: 250, want: 100},
	}
	for _, tt := range tests {
		if got := normalizeSearchLimit(tt.raw); got != tt.want {
			t.Fatalf("normalizeSearchLimit(%d) = %d, want %d", tt.raw, got, tt.want)
		}
	}
}

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

func mustParseSearchFilter(t *testing.T, raw map[string]any) filterExpression {
	t.Helper()

	expr, err := parseSearchFilter(raw)
	if err != nil {
		t.Fatalf("parseSearchFilter: %v", err)
	}
	return expr
}

func requireParsedComparison(t *testing.T, expr filterExpression, field string, op filterOperator) {
	t.Helper()

	for _, cmp := range flattenComparisons(expr) {
		if cmp.Field == field && cmp.Operator == op {
			return
		}
	}
	t.Fatalf("comparison %s %s not found in %#v", field, op, expr)
}

func containsWildcard(expr filterExpression) bool {
	for _, cmp := range flattenComparisons(expr) {
		for _, value := range cmp.Values {
			if value == "*" {
				return true
			}
		}
	}
	return false
}

package searchfilter_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/TrebuchetDynamics/goncho/internal/searchfilter"
)

func TestGrammarParsesHonchoOperators(t *testing.T) {
	expr, err := searchfilter.Parse(map[string]any{
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
		t.Fatalf("Parse: %v", err)
	}

	if expr.Kind != searchfilter.KindAnd {
		t.Fatalf("root kind = %v, want %v", expr.Kind, searchfilter.KindAnd)
	}
	requireParsedComparison(t, expr, "session_id", searchfilter.OpEQ)
	requireParsedComparison(t, expr, "created_at", searchfilter.OpGT)
	requireParsedComparison(t, expr, "created_at", searchfilter.OpGTE)
	requireParsedComparison(t, expr, "created_at", searchfilter.OpLT)
	requireParsedComparison(t, expr, "created_at", searchfilter.OpLTE)
	requireParsedComparison(t, expr, "created_at", searchfilter.OpNE)
	requireParsedComparison(t, expr, "peer_id", searchfilter.OpIn)
	requireParsedComparison(t, expr, "content", searchfilter.OpContains)
	requireParsedComparison(t, expr, "content", searchfilter.OpIContains)
	requireParsedComparison(t, expr, "metadata.profile.department", searchfilter.OpEQ)
	requireParsedComparison(t, expr, "metadata.score", searchfilter.OpGT)
	if !containsWildcard(expr) {
		t.Fatalf("parsed expression %#v does not preserve wildcard value", expr)
	}
}

func TestGrammarRejectsUnknownFieldsAndOperators(t *testing.T) {
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
			_, err := searchfilter.Parse(tt.filter)
			var unsupported *searchfilter.UnsupportedFilterError
			if !errors.As(err, &unsupported) {
				t.Fatalf("Parse err = %T %[1]v, want UnsupportedFilterError", err)
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

func TestCompilerSupportsSessionSourcePeerAndRejectsMetadata(t *testing.T) {
	supported, err := searchfilter.Compile(mustParse(t, map[string]any{
		"AND": []any{
			map[string]any{"session_id": map[string]any{"in": []any{"sess-discord", "*"}}},
			map[string]any{"source": "discord"},
			map[string]any{"peer_id": "user-juan"},
		},
	}), "user-juan")
	if err != nil {
		t.Fatalf("Compile supported subset: %v", err)
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

	unsupportedExpr := mustParse(t, map[string]any{
		"metadata": map[string]any{"priority": "high"},
	})
	_, err = searchfilter.Compile(unsupportedExpr, "user-juan")
	var unsupported *searchfilter.UnsupportedFilterError
	if !errors.As(err, &unsupported) {
		t.Fatalf("Compile metadata err = %T %[1]v, want UnsupportedFilterError", err)
	}
	if unsupported.Field != "metadata.priority" {
		t.Fatalf("UnsupportedFilterError.Field = %q, want metadata.priority", unsupported.Field)
	}
}

func TestNormalizeLimitDefaultsToTenAndClampsAtHonchoMaximum(t *testing.T) {
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
		if got := searchfilter.NormalizeLimit(tt.raw); got != tt.want {
			t.Fatalf("NormalizeLimit(%d) = %d, want %d", tt.raw, got, tt.want)
		}
	}
}

func mustParse(t *testing.T, raw map[string]any) searchfilter.Expression {
	t.Helper()

	expr, err := searchfilter.Parse(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return expr
}

func requireParsedComparison(t *testing.T, expr searchfilter.Expression, field string, op searchfilter.Operator) {
	t.Helper()

	for _, cmp := range searchfilter.FlattenComparisons(expr) {
		if cmp.Field == field && cmp.Operator == op {
			return
		}
	}
	t.Fatalf("comparison %s %s not found in %#v", field, op, expr)
}

func containsWildcard(expr searchfilter.Expression) bool {
	for _, cmp := range searchfilter.FlattenComparisons(expr) {
		for _, value := range cmp.Values {
			if value == "*" {
				return true
			}
		}
	}
	return false
}

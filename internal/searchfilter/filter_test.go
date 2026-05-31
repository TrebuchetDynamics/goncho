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
			map[string]any{"session_id": map[string]any{"eq": "sess-discord"}},
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
	if got := comparisonValues(t, expr, "session_id", searchfilter.OpEQ); !slices.Equal(got, []string{"sess-discord"}) {
		t.Fatalf("session_id eq values = %v, want sess-discord", got)
	}
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
		{
			name: "mixed metadata operator map",
			filter: map[string]any{"metadata": map[string]any{
				"score": map[string]any{"gt": 0.8, "regex": "^0\\."},
			}},
			wantField: "metadata.score",
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

func TestGrammarRejectsEmptyOperatorMaps(t *testing.T) {
	tests := []struct {
		name      string
		filter    map[string]any
		wantField string
	}{
		{
			name:      "top-level field",
			filter:    map[string]any{"session_id": map[string]any{}},
			wantField: "session_id",
		},
		{
			name: "metadata field",
			filter: map[string]any{"metadata": map[string]any{
				"score": map[string]any{},
			}},
			wantField: "metadata.score",
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
			if unsupported.Reason == "" {
				t.Fatalf("UnsupportedFilterError.Reason is empty for %+v", unsupported)
			}
		})
	}
}

func TestGrammarReportsFirstUnsupportedFilterDeterministically(t *testing.T) {
	for i := 0; i < 50; i++ {
		_, err := searchfilter.Parse(map[string]any{
			"zz_workspace_slug": "prod",
			"aa_workspace_id":   "workspace-1",
		})
		var unsupported *searchfilter.UnsupportedFilterError
		if !errors.As(err, &unsupported) {
			t.Fatalf("Parse err = %T %[1]v, want UnsupportedFilterError", err)
		}
		if unsupported.Field != "aa_workspace_id" {
			t.Fatalf("iteration %d unsupported field = %q, want deterministic lexicographic first field aa_workspace_id", i, unsupported.Field)
		}
	}
}

func TestGrammarReportsFirstUnknownOperatorDeterministically(t *testing.T) {
	for i := 0; i < 50; i++ {
		_, err := searchfilter.Parse(map[string]any{
			"created_at": map[string]any{
				"regex": "2024",
				"glob":  "2024*",
			},
		})
		var unsupported *searchfilter.UnsupportedFilterError
		if !errors.As(err, &unsupported) {
			t.Fatalf("Parse err = %T %[1]v, want UnsupportedFilterError", err)
		}
		if unsupported.Operator != "glob" {
			t.Fatalf("iteration %d unsupported operator = %q, want deterministic lexicographic first operator glob", i, unsupported.Operator)
		}
	}
}

func TestGrammarAcceptsTypedGoSlicesForInOperator(t *testing.T) {
	expr := mustParse(t, map[string]any{
		"session_id": map[string]any{"in": []string{"sess-discord", "sess-slack"}},
	})

	if got := comparisonValues(t, expr, "session_id", searchfilter.OpIn); !slices.Equal(got, []string{"sess-discord", "sess-slack"}) {
		t.Fatalf("session_id in values = %#v, want typed Go slice values preserved", got)
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

func TestCompilerTreatsBlankEqualityFiltersAsDenyAll(t *testing.T) {
	tests := []struct {
		name   string
		filter map[string]any
	}{
		{name: "blank session", filter: map[string]any{"session_id": "   "}},
		{name: "blank source", filter: map[string]any{"source": "\t"}},
		{name: "nil session", filter: map[string]any{"session_id": nil}},
		{name: "nil source", filter: map[string]any{"source": nil}},
		{name: "blank-only session in list", filter: map[string]any{"session_id": map[string]any{"in": []any{" ", ""}}}},
		{name: "blank-and-nil-only session in list", filter: map[string]any{"session_id": map[string]any{"in": []any{" ", nil, ""}}}},
		{name: "blank-only source in list", filter: map[string]any{"source": map[string]any{"in": []any{" ", ""}}}},
		{name: "blank-and-nil-only source in list", filter: map[string]any{"source": map[string]any{"in": []any{" ", nil, ""}}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiled, err := searchfilter.Compile(mustParse(t, tt.filter), "user-juan")
			if err != nil {
				t.Fatalf("Compile: %v", err)
			}
			if !compiled.DenyAll {
				t.Fatalf("DenyAll = false for blank-only equality filter: %+v", compiled)
			}
		})
	}
}

func TestCompilerTracksDenyAllWithoutReservedValueCollision(t *testing.T) {
	literal, err := searchfilter.Compile(mustParse(t, map[string]any{
		"session_id": "__deny_all__",
	}), "user-juan")
	if err != nil {
		t.Fatalf("Compile literal reserved-looking session_id: %v", err)
	}
	if literal.DenyAll {
		t.Fatal("DenyAll = true for literal __deny_all__ session_id")
	}
	if !slices.Equal(literal.SessionIDs, []string{"__deny_all__"}) {
		t.Fatalf("SessionIDs = %#v, want literal __deny_all__", literal.SessionIDs)
	}

	contradiction, err := searchfilter.Compile(mustParse(t, map[string]any{
		"AND": []any{
			map[string]any{"session_id": "sess-a"},
			map[string]any{"session_id": "sess-b"},
		},
	}), "user-juan")
	if err != nil {
		t.Fatalf("Compile contradictory session filters: %v", err)
	}
	if !contradiction.DenyAll {
		t.Fatalf("DenyAll = false for contradictory session filters: %+v", contradiction)
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

func TestHasWildcardTrimsValuesLikeSourceFilters(t *testing.T) {
	if !searchfilter.HasWildcard([]string{" * "}) {
		t.Fatalf("HasWildcard should detect a wildcard after trimming source values")
	}
	if searchfilter.HasWildcard([]string{"", "source"}) {
		t.Fatalf("HasWildcard should not treat blank values as wildcard in compiled search filters")
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

	_ = comparisonValues(t, expr, field, op)
}

func comparisonValues(t *testing.T, expr searchfilter.Expression, field string, op searchfilter.Operator) []string {
	t.Helper()

	for _, cmp := range searchfilter.FlattenComparisons(expr) {
		if cmp.Field == field && cmp.Operator == op {
			return cmp.Values
		}
	}
	t.Fatalf("comparison %s %s not found in %#v", field, op, expr)
	return nil
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

package sessionquery

import (
	"reflect"
	"strings"
	"testing"
)

func TestBuildTurnSearchQuerySanitizesFTSAndNormalizesRoles(t *testing.T) {
	query, args := BuildTurnSearchQuery(` owner:(alice) `, []string{"s1"}, []string{"claude:c1"}, []string{" User ", "user", "assistant"}, 5, false)

	for _, fragment := range []string{
		`SELECT t.session_id, t.chat_id, t.role, t.content, t.ts_unix FROM turns t`,
		`JOIN turns_fts fts ON fts.rowid = t.id WHERE turns_fts MATCH ?`,
		`t.session_id IN (?) OR t.chat_id IN (?)`,
		`t.memory_sync_status = 'ready'`,
		`t.role IN (?,?)`,
		`ORDER BY t.ts_unix DESC, t.id DESC LIMIT ?`,
	} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("query missing %q in %q", fragment, query)
		}
	}
	wantArgs := []any{"owner  alice", "s1", "claude:c1", "user", "assistant", 5}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", args, wantArgs)
	}
}

func TestBuildTurnSearchQuerySessionsOnlyOmitsBlankRoleFilter(t *testing.T) {
	query, args := BuildTurnSearchQuery("", []string{"s1", "s2"}, nil, []string{" "}, 10, true)

	if !strings.Contains(query, `SELECT t.session_id, t.chat_id, MAX(t.ts_unix) AS latest_turn_unix FROM turns t`) {
		t.Fatalf("query does not select sessions: %q", query)
	}
	if strings.Contains(query, `t.role IN`) {
		t.Fatalf("query should omit blank role filter: %q", query)
	}
	if !strings.Contains(query, `GROUP BY t.session_id, t.chat_id ORDER BY latest_turn_unix DESC, t.session_id ASC LIMIT ?`) {
		t.Fatalf("query missing session ordering: %q", query)
	}
	wantArgs := []any{"s1", "s2", 10}
	if !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("args = %#v, want %#v", args, wantArgs)
	}
}

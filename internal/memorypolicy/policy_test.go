package memorypolicy

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
)

func TestTierValidationAndDefaults(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"global", true}, {"project", true}, {"task", true},
		{"workspace", true}, {"decision", true},
		{"GLOBAL", true}, {" Project ", true},
		{"", false}, {"invalid", false}, {"admin", false},
	}
	for _, tc := range tests {
		got := ValidTier(tc.input)
		if got != tc.want {
			t.Errorf("ValidTier(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}

	if got := NormalizeTier(""); got != TierGlobal {
		t.Errorf("empty -> %v, want %v", got, TierGlobal)
	}
	if got := NormalizeTier("project"); got != TierProject {
		t.Errorf("project -> %v, want %v", got, TierProject)
	}
	if got := NormalizeTier("UNKNOWN"); got != TierGlobal {
		t.Errorf("unknown -> %v, want %v", got, TierGlobal)
	}

	if got := DefaultTierForSource("manual"); got != TierProject {
		t.Errorf("manual -> %v, want project", got)
	}
	if got := DefaultTierForSource("runtime"); got != TierTask {
		t.Errorf("runtime -> %v, want task", got)
	}
	if got := DefaultTierForSource("reviewed_proposal"); got != TierProject {
		t.Errorf("reviewed_proposal -> %v, want project", got)
	}
}

func TestReadableAndWritableTiers(t *testing.T) {
	tiers := ReadableBy(TierTask)
	if len(tiers) != 3 {
		t.Fatalf("Task agent should read 3 tiers, got %d", len(tiers))
	}
	expected := []Tier{TierGlobal, TierProject, TierTask}
	for i, want := range expected {
		if tiers[i] != want {
			t.Errorf("tiers[%d] = %v, want %v", i, tiers[i], want)
		}
	}

	childTiers := WritableBy(false)
	if len(childTiers) != 1 || childTiers[0] != TierWorkspace {
		t.Errorf("child writable tiers = %v, want [workspace]", childTiers)
	}
	parentTiers := WritableBy(true)
	if len(parentTiers) != 5 {
		t.Errorf("parent writable tier count = %d, want 5", len(parentTiers))
	}
}

func TestACLQueryReadScopeTierBased(t *testing.T) {
	db := setupACLTestDB(t)
	defer db.Close()
	ctx := context.Background()
	seedMemory(t, db, "mem1", "parent", "ws1", "project", "project-level fact")
	seedMemory(t, db, "mem2", "parent", "ws1", "global", "global fact")
	seedMemory(t, db, "mem3", "child1", "ws1", "workspace", "child scratch")
	seedMemory(t, db, "mem4", "parent", "ws2", "project", "other workspace")
	q := ACLQuery{AgentID: "child1", IsParent: false, ReadTiers: []Tier{TierGlobal, TierProject, TierTask}, WorkspaceID: "ws1"}
	clause, args := q.ReadScopeSQL()
	query := `SELECT m.memory_id, m.content FROM goncho_memory_items m WHERE m.active = 1 AND ` + clause + ` ORDER BY m.memory_id`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id, content string
		rows.Scan(&id, &content)
		ids = append(ids, id)
	}
	seen := make(map[string]bool)
	for _, id := range ids {
		seen[id] = true
	}
	if !seen["mem1"] {
		t.Error("child should see project-tier memory in its workspace")
	}
	if !seen["mem2"] {
		t.Error("child should see global-tier memory")
	}
	if !seen["mem3"] {
		t.Error("child should see its own workspace memory")
	}
	if seen["mem4"] {
		t.Error("child should NOT see memory in other workspace")
	}
}

func TestACLQueryExplicitGrant(t *testing.T) {
	db := setupACLTestDB(t)
	defer db.Close()
	ctx := context.Background()
	seedMemory(t, db, "mem-decision", "parent", "ws1", "decision", "strategic decision")
	seedACL(t, db, "mem-decision", "child2", "read", "parent")
	q := ACLQuery{AgentID: "child2", IsParent: false, ReadTiers: []Tier{TierGlobal, TierProject}, WorkspaceID: "ws1"}
	ok, err := AgentCanAccessMemory(ctx, db, "mem-decision", "child2", "ws1", q.ReadTiers)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("child2 should read mem-decision via explicit ACL grant")
	}
}

func TestACLQueryChildCannotWriteDecisionTier(t *testing.T) {
	q := ACLQuery{AgentID: "child1", IsParent: false, WriteTier: TierWorkspace}
	if q.CanWrite(TierDecision) {
		t.Error("child should not be able to write to decision tier")
	}
	if !q.CanWrite(TierWorkspace) {
		t.Error("child should be able to write to workspace tier")
	}
}

func TestACLQueryParentCanWriteAll(t *testing.T) {
	q := ACLQuery{IsParent: true}
	for _, tier := range Hierarchy() {
		if !q.CanWrite(tier) {
			t.Errorf("parent should be able to write to tier %s", tier)
		}
	}
}

func setupACLTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.Exec(`CREATE TABLE IF NOT EXISTS goncho_memory_items(memory_id TEXT PRIMARY KEY, contract_version TEXT DEFAULT '1', agent_id TEXT, workspace_id TEXT, observer_peer_id TEXT, peer_id TEXT, session_key TEXT DEFAULT '', source_kind TEXT, content TEXT, revision INTEGER DEFAULT 1, active INTEGER DEFAULT 1, tombstoned_at INTEGER, tombstone_reason TEXT, scope TEXT DEFAULT 'private', tier TEXT DEFAULT 'global' CHECK(tier IN ('global','project','task','workspace','decision')), provenance_json TEXT DEFAULT '{}', tags_json TEXT DEFAULT '[]', importance REAL DEFAULT 0.5, created_at INTEGER, updated_at INTEGER)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS memory_acl(id INTEGER PRIMARY KEY AUTOINCREMENT, memory_id TEXT REFERENCES goncho_memory_items(memory_id) ON DELETE CASCADE, agent_id TEXT, permission TEXT CHECK(permission IN ('read','propose','write')), granted_by TEXT, granted_at INTEGER, UNIQUE(memory_id, agent_id, permission))`)
	return db
}

func seedMemory(t *testing.T, db *sql.DB, id, agentID, wsID, tier, content string) {
	t.Helper()
	now := time.Now().Unix()
	db.Exec(`INSERT OR REPLACE INTO goncho_memory_items(memory_id, agent_id, workspace_id, observer_peer_id, peer_id, source_kind, content, tier, scope, created_at, updated_at) VALUES(?, ?, ?, 'obs', 'peer', 'manual', ?, ?, 'private', ?, ?)`, id, agentID, wsID, content, tier, now, now)
}

func seedACL(t *testing.T, db *sql.DB, memID, agentID, perm, grantedBy string) {
	t.Helper()
	db.Exec(`INSERT OR IGNORE INTO memory_acl(memory_id, agent_id, permission, granted_by, granted_at) VALUES(?, ?, ?, ?, unixepoch())`, memID, agentID, perm, grantedBy)
}

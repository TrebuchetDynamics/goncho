package goncho

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
)

func TestMemoryPolicyPublicFacadeAppliesExplicitACLGrant(t *testing.T) {
	db := setupMemoryPolicyFacadeTestDB(t)
	defer db.Close()
	ctx := context.Background()
	seedMemoryPolicyFacadeMemory(t, db, "mem-decision", "parent", "ws1", string(TierDecision), "strategic decision")
	if err := GrantReadACL(ctx, db, "mem-decision", "child2", "parent"); err != nil {
		t.Fatal(err)
	}

	q := ACLQuery{AgentID: "child2", IsParent: false, ReadTiers: []MemoryTier{TierGlobal, TierProject}, WorkspaceID: "ws1"}
	ok, err := AgentCanAccessMemory(ctx, db, "mem-decision", "child2", "ws1", q.ReadTiers)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("child2 should read mem-decision via explicit ACL grant through public facade")
	}
	if q.CanWrite(TierDecision) {
		t.Fatal("child2 should not write decision tier through public facade")
	}
}

func setupMemoryPolicyFacadeTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.Exec(`CREATE TABLE IF NOT EXISTS goncho_memory_items(memory_id TEXT PRIMARY KEY, contract_version TEXT DEFAULT '1', agent_id TEXT, workspace_id TEXT, observer_peer_id TEXT, peer_id TEXT, session_key TEXT DEFAULT '', source_kind TEXT, content TEXT, revision INTEGER DEFAULT 1, active INTEGER DEFAULT 1, tombstoned_at INTEGER, tombstone_reason TEXT, scope TEXT DEFAULT 'private', tier TEXT DEFAULT 'global' CHECK(tier IN ('global','project','task','workspace','decision')), provenance_json TEXT DEFAULT '{}', tags_json TEXT DEFAULT '[]', importance REAL DEFAULT 0.5, created_at INTEGER, updated_at INTEGER)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS memory_acl(id INTEGER PRIMARY KEY AUTOINCREMENT, memory_id TEXT REFERENCES goncho_memory_items(memory_id) ON DELETE CASCADE, agent_id TEXT, permission TEXT CHECK(permission IN ('read','propose','write')), granted_by TEXT, granted_at INTEGER, UNIQUE(memory_id, agent_id, permission))`)
	return db
}

func seedMemoryPolicyFacadeMemory(t *testing.T, db *sql.DB, id, agentID, wsID, tier, content string) {
	t.Helper()
	now := time.Now().Unix()
	if _, err := db.Exec(`INSERT OR REPLACE INTO goncho_memory_items(memory_id, agent_id, workspace_id, observer_peer_id, peer_id, source_kind, content, tier, scope, created_at, updated_at) VALUES(?, ?, ?, 'obs', 'peer', 'manual', ?, ?, 'private', ?, ?)`, id, agentID, wsID, content, tier, now, now); err != nil {
		t.Fatal(err)
	}
}

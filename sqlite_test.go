package goncho

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/ncruces/go-sqlite3/driver"
)

type TestSqliteStore struct {
	db *sql.DB
}

func OpenTestSqlite(path string) (*TestSqliteStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("goncho: create parent dir: %w", err)
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("goncho: open %s: %w", path, err)
	}
	db.SetMaxOpenConns(1)

	pragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA busy_timeout = 2000",
		"PRAGMA foreign_keys = ON",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("%s: %w", p, err)
		}
	}

	if err := ensureGonchoSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &TestSqliteStore{db: db}, nil
}

func (t *TestSqliteStore) DB() *sql.DB    { return t.db }
func (t *TestSqliteStore) Close(ctx context.Context) error { return t.db.Close() }

func ensureGonchoSchema(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS turns (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			role TEXT NOT NULL CHECK(role IN ('user','assistant')),
			content TEXT NOT NULL,
			ts_unix INTEGER NOT NULL,
			chat_id TEXT NOT NULL DEFAULT '',
			meta_json TEXT,
			turn_key TEXT,
			memory_sync_status TEXT NOT NULL DEFAULT 'ready' CHECK(memory_sync_status IN ('pending','ready','skipped')),
			memory_sync_reason TEXT CHECK(memory_sync_reason IS NULL OR memory_sync_reason IN ('interrupted','cancelled','client_disconnect')),
			extracted INTEGER NOT NULL DEFAULT 0,
			extraction_attempts INTEGER NOT NULL DEFAULT 0,
			extraction_error TEXT,
			cron INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_turns_session_ts ON turns(session_id, ts_unix)`,
		`CREATE INDEX IF NOT EXISTS idx_turns_unextracted ON turns(id) WHERE extracted = 0`,
		`CREATE INDEX IF NOT EXISTS idx_turns_memory_sync ON turns(memory_sync_status, extracted, cron, id)`,
		`CREATE INDEX IF NOT EXISTS idx_turns_turn_key ON turns(turn_key) WHERE turn_key IS NOT NULL`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS turns_fts USING fts5(content, content='turns', content_rowid='id')`,
		`CREATE TABLE IF NOT EXISTS goncho_peer_cards (
			workspace_id TEXT NOT NULL,
			observer_peer_id TEXT NOT NULL,
			peer_id TEXT NOT NULL,
			card_json TEXT NOT NULL,
			updated_at INTEGER NOT NULL,
			PRIMARY KEY(workspace_id, observer_peer_id, peer_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_goncho_peer_cards_observed ON goncho_peer_cards(workspace_id, peer_id, updated_at DESC)`,
		`CREATE TABLE IF NOT EXISTS goncho_conclusions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			workspace_id TEXT NOT NULL,
			observer_peer_id TEXT NOT NULL,
			peer_id TEXT NOT NULL,
			session_key TEXT,
			content TEXT NOT NULL,
			kind TEXT NOT NULL DEFAULT 'manual',
			status TEXT NOT NULL CHECK(status IN ('pending','processed','dead_letter')),
			source TEXT NOT NULL DEFAULT 'manual',
			idempotency_key TEXT NOT NULL,
			evidence_json TEXT NOT NULL DEFAULT '[]',
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			scope TEXT NOT NULL DEFAULT 'workspace'
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_goncho_conclusions_idempotency ON goncho_conclusions(workspace_id, observer_peer_id, peer_id, idempotency_key)`,
		`CREATE INDEX IF NOT EXISTS idx_goncho_conclusions_peer ON goncho_conclusions(workspace_id, peer_id, updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_goncho_conclusions_session ON goncho_conclusions(workspace_id, session_key, updated_at DESC)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS goncho_conclusions_fts USING fts5(content, content='goncho_conclusions', content_rowid='id')`,
		`CREATE TABLE IF NOT EXISTS goncho_session_summaries (
			workspace_id TEXT NOT NULL,
			session_key TEXT NOT NULL,
			summary_type TEXT NOT NULL CHECK(summary_type IN ('short','long','structured')),
			content TEXT NOT NULL,
			message_id INTEGER NOT NULL,
			created_at INTEGER NOT NULL,
			token_count INTEGER NOT NULL CHECK(token_count >= 0),
			PRIMARY KEY(workspace_id, session_key, summary_type)
		)`,
		`CREATE TABLE IF NOT EXISTS goncho_memory_items (
			memory_id TEXT PRIMARY KEY,
			contract_version TEXT NOT NULL DEFAULT '1',
			agent_id TEXT NOT NULL,
			workspace_id TEXT NOT NULL,
			observer_peer_id TEXT NOT NULL,
			peer_id TEXT NOT NULL,
			session_key TEXT DEFAULT '',
			source_kind TEXT,
			content TEXT,
			revision INTEGER NOT NULL DEFAULT 1,
			active INTEGER NOT NULL DEFAULT 1,
			tombstoned_at INTEGER,
			tombstone_reason TEXT,
			scope TEXT NOT NULL DEFAULT 'private',
			tier TEXT NOT NULL DEFAULT 'global' CHECK(tier IN ('global','project','task','workspace','decision')),
			provenance_json TEXT NOT NULL DEFAULT '{}',
			tags_json TEXT NOT NULL DEFAULT '[]',
			importance REAL NOT NULL DEFAULT 0.5,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS goncho_memory_fts USING fts5(content, content='goncho_memory_items', content_rowid='rowid')`,
		`CREATE TABLE IF NOT EXISTS goncho_dreams (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			workspace_id TEXT NOT NULL,
			observer_peer_id TEXT NOT NULL,
			observed_peer_id TEXT NOT NULL,
			work_unit_key TEXT NOT NULL,
			dream_type TEXT NOT NULL DEFAULT 'consolidation',
			status TEXT NOT NULL CHECK(status IN ('pending','in_progress','completed','stale','cancelled','rejected')),
			manual INTEGER NOT NULL DEFAULT 0 CHECK(manual IN (0,1)),
			reason TEXT NOT NULL,
			new_conclusions INTEGER NOT NULL DEFAULT 0 CHECK(new_conclusions >= 0),
			min_conclusions INTEGER NOT NULL DEFAULT 50 CHECK(min_conclusions >= 0),
			last_conclusion_id INTEGER NOT NULL DEFAULT 0,
			scheduled_for INTEGER NOT NULL,
			started_at INTEGER,
			completed_at INTEGER,
			cancelled_at INTEGER,
			stale_at INTEGER,
			last_activity_at INTEGER NOT NULL DEFAULT 0,
			cooldown_until INTEGER NOT NULL DEFAULT 0,
			idle_until INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_goncho_dreams_active_scope ON goncho_dreams(workspace_id, observer_peer_id, observed_peer_id) WHERE status IN ('pending','in_progress')`,
		`CREATE INDEX IF NOT EXISTS idx_goncho_dreams_scope_updated ON goncho_dreams(workspace_id, observer_peer_id, observed_peer_id, updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_goncho_dreams_status ON goncho_dreams(workspace_id, observer_peer_id, status, updated_at DESC)`,
		`CREATE TABLE IF NOT EXISTS goncho_dynamic_agents (
			id TEXT PRIMARY KEY CHECK(length(id) BETWEEN 1 AND 64),
			name TEXT NOT NULL,
			persona TEXT NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_goncho_dynamic_agents_created ON goncho_dynamic_agents(created_at)`,
		`CREATE TABLE IF NOT EXISTS goncho_dynamic_agent_bindings (
			channel TEXT NOT NULL,
			peer_kind TEXT NOT NULL,
			peer_id TEXT NOT NULL,
			thread_id TEXT NOT NULL DEFAULT '',
			agent_id TEXT NOT NULL,
			bound_at INTEGER NOT NULL,
			PRIMARY KEY(channel, peer_kind, peer_id, thread_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_goncho_dynamic_agent_bindings_agent ON goncho_dynamic_agent_bindings(agent_id)`,
		`CREATE TABLE IF NOT EXISTS goncho_webhook_endpoints (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL,
			url TEXT NOT NULL CHECK(length(url) <= 2048),
			created_at INTEGER NOT NULL,
			UNIQUE(workspace_id, url)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_goncho_webhook_endpoints_workspace ON goncho_webhook_endpoints(workspace_id, created_at)`,
		`CREATE TABLE IF NOT EXISTS memory_acl (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			memory_id TEXT REFERENCES goncho_memory_items(memory_id) ON DELETE CASCADE,
			agent_id TEXT,
			permission TEXT CHECK(permission IN ('read','propose','write')),
			granted_by TEXT,
			granted_at INTEGER,
			UNIQUE(memory_id, agent_id, permission)
		)`,
		`CREATE TABLE IF NOT EXISTS memory_proposals (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			memory_id TEXT,
			agent_id TEXT,
			content TEXT,
			status TEXT DEFAULT 'pending',
			created_at INTEGER,
			updated_at INTEGER
		)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("goncho schema: %w", err)
		}
	}
	return nil
}

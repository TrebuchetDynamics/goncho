package memory

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/TrebuchetDynamics/goncho/internal/memoryannotations"
	_ "github.com/ncruces/go-sqlite3/driver"
)

const (
	CrossChatDecisionAllowed  = "allowed"
	CrossChatDecisionDenied   = "denied"
	CrossChatDecisionDegraded = "degraded"
	CrossChatFallbackSameChat = "same-chat"
)

type SqliteStore struct {
	db *sql.DB
}

func OpenSqlite(path string, _ int, _ *slog.Logger) (*SqliteStore, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("memory: create parent dir: %w", err)
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("memory: open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	if err := applyPragmas(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := ensureSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &SqliteStore{db: db}, nil
}

func (s *SqliteStore) DB() *sql.DB { return s.db }

func (s *SqliteStore) Close(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func applyPragmas(db *sql.DB) error {
	for _, stmt := range []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA busy_timeout = 2000",
		"PRAGMA foreign_keys = ON",
	} {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("memory: %s: %w", stmt, err)
		}
	}
	return nil
}

func ensureSchema(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS turns (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL DEFAULT '',
			role TEXT NOT NULL DEFAULT '',
			content TEXT NOT NULL DEFAULT '',
			ts_unix INTEGER NOT NULL DEFAULT 0,
			chat_id TEXT,
			meta_json TEXT,
			memory_sync_status TEXT NOT NULL DEFAULT 'ready',
			memory_sync_reason TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_turns_session ON turns(session_id, ts_unix DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_turns_chat ON turns(chat_id, ts_unix DESC, id DESC)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS turns_fts USING fts5(content, content='turns', content_rowid='id')`,
		`CREATE TRIGGER IF NOT EXISTS turns_fts_ai AFTER INSERT ON turns BEGIN
			INSERT INTO turns_fts(rowid, content) VALUES (new.id, new.content);
		END`,
		`CREATE TRIGGER IF NOT EXISTS turns_fts_ad AFTER DELETE ON turns BEGIN
			INSERT INTO turns_fts(turns_fts, rowid, content) VALUES('delete', old.id, old.content);
		END`,
		`CREATE TRIGGER IF NOT EXISTS turns_fts_au AFTER UPDATE OF content ON turns BEGIN
			INSERT INTO turns_fts(turns_fts, rowid, content) VALUES('delete', old.id, old.content);
			INSERT INTO turns_fts(rowid, content) VALUES (new.id, new.content);
		END`,
		`CREATE TABLE IF NOT EXISTS goncho_peer_cards (
			workspace_id TEXT NOT NULL,
			profile_id TEXT NOT NULL DEFAULT '',
			observer_peer_id TEXT NOT NULL,
			peer_id TEXT NOT NULL,
			card_json TEXT NOT NULL DEFAULT '[]',
			updated_at INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY(workspace_id, profile_id, observer_peer_id, peer_id)
		)`,
		`ALTER TABLE goncho_peer_cards ADD COLUMN profile_id TEXT NOT NULL DEFAULT ''`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_goncho_peer_cards_profile_identity ON goncho_peer_cards(workspace_id, profile_id, observer_peer_id, peer_id)`,
		`CREATE TABLE IF NOT EXISTS goncho_conclusions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			workspace_id TEXT NOT NULL,
			profile_id TEXT NOT NULL DEFAULT '',
			observer_peer_id TEXT NOT NULL,
			peer_id TEXT NOT NULL,
			session_key TEXT,
			content TEXT NOT NULL,
			kind TEXT NOT NULL DEFAULT 'fact',
			status TEXT NOT NULL DEFAULT 'active',
			source TEXT NOT NULL DEFAULT 'manual',
			idempotency_key TEXT NOT NULL,
			evidence_json TEXT NOT NULL DEFAULT '{}',
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			scope TEXT NOT NULL DEFAULT 'workspace',
			UNIQUE(workspace_id, profile_id, observer_peer_id, peer_id, idempotency_key)
		)`,
		`ALTER TABLE goncho_conclusions ADD COLUMN profile_id TEXT NOT NULL DEFAULT ''`,
		`CREATE INDEX IF NOT EXISTS idx_goncho_conclusions_lookup ON goncho_conclusions(workspace_id, profile_id, observer_peer_id, peer_id, session_key, updated_at DESC)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_goncho_conclusions_profile_idempotency ON goncho_conclusions(workspace_id, profile_id, observer_peer_id, peer_id, idempotency_key)`,
		`CREATE TABLE IF NOT EXISTS goncho_memory_annotations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			workspace_id TEXT NOT NULL,
			profile_id TEXT NOT NULL DEFAULT '',
			observer_peer_id TEXT NOT NULL,
			peer_id TEXT NOT NULL,
			memory_source TEXT NOT NULL,
			memory_id INTEGER NOT NULL,
			kind TEXT NOT NULL,
			value TEXT NOT NULL,
			source TEXT NOT NULL DEFAULT '',
			confidence REAL NOT NULL DEFAULT 1.0,
			created_at INTEGER NOT NULL,
			FOREIGN KEY(memory_id) REFERENCES goncho_conclusions(id) ON DELETE CASCADE,
			UNIQUE(workspace_id, profile_id, observer_peer_id, peer_id, memory_source, memory_id, kind, value)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_goncho_memory_annotations_memory_kind ON goncho_memory_annotations(workspace_id, profile_id, observer_peer_id, peer_id, memory_source, memory_id, kind)`,
		`CREATE INDEX IF NOT EXISTS idx_goncho_memory_annotations_kind_value ON goncho_memory_annotations(kind, value)`,
		`CREATE TABLE IF NOT EXISTS goncho_session_summaries (
			workspace_id TEXT NOT NULL,
			session_key TEXT NOT NULL,
			summary_type TEXT NOT NULL,
			content TEXT NOT NULL,
			message_id INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL DEFAULT 0,
			token_count INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY(workspace_id, session_key, summary_type)
		)`,
		`CREATE TABLE IF NOT EXISTS goncho_dreams (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			workspace_id TEXT NOT NULL,
			observer_peer_id TEXT NOT NULL,
			observed_peer_id TEXT NOT NULL,
			dream_type TEXT NOT NULL DEFAULT 'consolidation',
			status TEXT NOT NULL,
			manual INTEGER NOT NULL DEFAULT 0,
			work_unit_key TEXT NOT NULL,
			reason TEXT NOT NULL DEFAULT '',
			new_conclusions INTEGER NOT NULL DEFAULT 0,
			min_conclusions INTEGER NOT NULL DEFAULT 0,
			last_conclusion_id INTEGER NOT NULL DEFAULT 0,
			scheduled_for INTEGER NOT NULL DEFAULT 0,
			last_activity_at INTEGER NOT NULL DEFAULT 0,
			cooldown_until INTEGER NOT NULL DEFAULT 0,
			idle_until INTEGER NOT NULL DEFAULT 0,
			started_at INTEGER,
			completed_at INTEGER,
			stale_at INTEGER,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS goncho_memory_items (
			memory_id TEXT PRIMARY KEY,
			contract_version TEXT NOT NULL DEFAULT '1',
			agent_id TEXT NOT NULL DEFAULT '',
			workspace_id TEXT NOT NULL DEFAULT '',
			observer_peer_id TEXT NOT NULL DEFAULT '',
			peer_id TEXT NOT NULL DEFAULT '',
			session_key TEXT NOT NULL DEFAULT '',
			source_kind TEXT NOT NULL DEFAULT '',
			content TEXT NOT NULL DEFAULT '',
			revision INTEGER NOT NULL DEFAULT 1,
			active INTEGER NOT NULL DEFAULT 1,
			tombstoned_at INTEGER,
			tombstone_reason TEXT NOT NULL DEFAULT '',
			scope TEXT NOT NULL DEFAULT 'private',
			tier TEXT NOT NULL DEFAULT 'global',
			provenance_json TEXT NOT NULL DEFAULT '{}',
			tags_json TEXT NOT NULL DEFAULT '[]',
			importance REAL NOT NULL DEFAULT 0.5,
			created_at INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS goncho_memory_eval_artifacts (
			artifact_id TEXT PRIMARY KEY,
			agent_id TEXT NOT NULL DEFAULT '',
			workspace_id TEXT NOT NULL DEFAULT '',
			peer_id TEXT NOT NULL DEFAULT '',
			session_id TEXT NOT NULL DEFAULT '',
			type TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT '',
			source_memory_id TEXT NOT NULL DEFAULT '',
			shared INTEGER NOT NULL DEFAULT 0,
			payload_json TEXT NOT NULL DEFAULT '{}'
		)`,
	}
	statements = append(statements, memoryannotations.DDL...)
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
				continue
			}
			return fmt.Errorf("memory: apply schema: %w", err)
		}
	}
	return nil
}

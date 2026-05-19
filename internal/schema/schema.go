package schema

import (
	"database/sql"
	"fmt"
)

// RunMigrations creates all Goncho v2 tables and indexes if they do not exist.
// It also sets recommended PRAGMAs for SQLite performance and safety.
func RunMigrations(db *sql.DB) error {
	pragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA busy_timeout = 2000",
		"PRAGMA foreign_keys = ON",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return fmt.Errorf("goncho: pragma %s: %w", p, err)
		}
	}

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS memories (
			id TEXT PRIMARY KEY,
			kind TEXT NOT NULL CHECK(kind IN ('conclusion','memory','profile','summary','preference','fact','decision')),
			content TEXT NOT NULL,
			peer_id TEXT NOT NULL DEFAULT '',
			workspace_id TEXT NOT NULL DEFAULT '',
			scope TEXT NOT NULL DEFAULT 'private' CHECK(scope IN ('private','workspace','global','project','task')),
			context_id TEXT NOT NULL DEFAULT '',
			importance REAL NOT NULL DEFAULT 0.5 CHECK(importance >= 0 AND importance <= 1),
			valid_from INTEGER NOT NULL,
			valid_until INTEGER,
			supersedes_id TEXT REFERENCES memories(id),
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			checksum TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_memories_peer_workspace ON memories(peer_id, workspace_id, updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_memories_context ON memories(context_id, workspace_id, updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_memories_valid ON memories(valid_from, valid_until) WHERE valid_until IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_memories_supersedes ON memories(supersedes_id) WHERE supersedes_id IS NOT NULL`,

		`CREATE VIRTUAL TABLE IF NOT EXISTS memory_fts USING fts5(content, memory_id)`,

		`CREATE TABLE IF NOT EXISTS memory_relations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source_id TEXT NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
			target_entity TEXT NOT NULL,
			relation_type TEXT NOT NULL,
			confidence REAL NOT NULL DEFAULT 0.5 CHECK(confidence >= 0 AND confidence <= 1),
			created_at INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_relations_source ON memory_relations(source_id)`,
		`CREATE INDEX IF NOT EXISTS idx_relations_target ON memory_relations(target_entity)`,

		`CREATE TABLE IF NOT EXISTS goals (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','completed','archived')),
			parent_id TEXT REFERENCES goals(id),
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_goals_status ON goals(status, updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_goals_parent ON goals(parent_id)`,
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("goncho: begin migration tx: %w", err)
	}
	defer tx.Rollback()

	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("goncho: migration exec: %w", err)
		}
	}

	return tx.Commit()
}

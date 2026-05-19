package goncho

import (
	"database/sql"
	"fmt"
)

func RunMigrations(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("%w: nil db", ErrObservationInvalid)
	}
	for _, stmt := range []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA busy_timeout = 2000",
		"PRAGMA foreign_keys = ON",
	} {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("goncho: run migration pragma %q: %w", stmt, err)
		}
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("goncho: begin migrations: %w", err)
	}
	defer tx.Rollback()
	for _, stmt := range gonchoObservationDDL {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("goncho: apply observation migration: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("goncho: commit observation migrations: %w", err)
	}
	return nil
}

var gonchoObservationDDL = []string{
	`CREATE TABLE IF NOT EXISTS goncho_observations (
		id TEXT PRIMARY KEY,
		kind TEXT NOT NULL CHECK(kind IN ('session_start','user_prompt','tool_call','tool_result','tool_error','assistant_response','compact','session_end','custom')),
		workspace_id TEXT NOT NULL DEFAULT '',
		peer_id TEXT NOT NULL DEFAULT '',
		session_key TEXT NOT NULL DEFAULT '',
		context_id TEXT NOT NULL DEFAULT '',
		input TEXT NOT NULL DEFAULT '',
		output TEXT NOT NULL DEFAULT '',
		success INTEGER CHECK(success IN (0, 1) OR success IS NULL),
		metadata_json TEXT NOT NULL DEFAULT '{}',
		input_truncated INTEGER NOT NULL DEFAULT 0 CHECK(input_truncated IN (0, 1)),
		output_truncated INTEGER NOT NULL DEFAULT 0 CHECK(output_truncated IN (0, 1)),
		input_original_bytes INTEGER NOT NULL DEFAULT 0,
		output_original_bytes INTEGER NOT NULL DEFAULT 0,
		redacted INTEGER NOT NULL DEFAULT 0 CHECK(redacted IN (0, 1)),
		redaction_count INTEGER NOT NULL DEFAULT 0,
		checksum TEXT NOT NULL,
		observed_at INTEGER NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_observations_observed_at ON goncho_observations(observed_at DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_observations_workspace ON goncho_observations(workspace_id, observed_at DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_observations_peer ON goncho_observations(peer_id, observed_at DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_observations_session ON goncho_observations(session_key, observed_at DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_observations_context ON goncho_observations(context_id, observed_at DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_observations_kind ON goncho_observations(kind, observed_at DESC)`,
	`CREATE TABLE IF NOT EXISTS goncho_audit_events (
		id TEXT PRIMARY KEY,
		action TEXT NOT NULL CHECK(action IN ('observe')),
		target_type TEXT NOT NULL CHECK(target_type IN ('observation')),
		target_id TEXT NOT NULL,
		workspace_id TEXT NOT NULL DEFAULT '',
		peer_id TEXT NOT NULL DEFAULT '',
		session_key TEXT NOT NULL DEFAULT '',
		reason TEXT NOT NULL,
		metadata_json TEXT NOT NULL DEFAULT '{}',
		created_at INTEGER NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_audit_events_created_at ON goncho_audit_events(created_at DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_audit_events_target ON goncho_audit_events(target_type, target_id, created_at DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_audit_events_action ON goncho_audit_events(action, created_at DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_audit_events_workspace ON goncho_audit_events(workspace_id, created_at DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_audit_events_peer ON goncho_audit_events(peer_id, created_at DESC)`,
	`CREATE INDEX IF NOT EXISTS idx_goncho_audit_events_session ON goncho_audit_events(session_key, created_at DESC)`,
}

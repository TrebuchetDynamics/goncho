package memoryannotations

var DDL = []string{
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
}

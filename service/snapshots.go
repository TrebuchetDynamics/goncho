package goncho

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/dbscan"
	"github.com/TrebuchetDynamics/goncho/service/internal/hashutil"
)

type SnapshotParams struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	ProfileID   string `json:"profile_id,omitempty"`
	Peer        string `json:"peer"`
}

type SnapshotGitMetadata struct {
	AdapterOwned bool   `json:"adapter_owned"`
	Operation    string `json:"operation"`
	Note         string `json:"note"`
}

type SnapshotEntry struct {
	Kind     string            `json:"kind"`
	Key      string            `json:"key"`
	Checksum string            `json:"checksum"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type SnapshotManifest struct {
	ManifestVersion string              `json:"manifest_version"`
	SnapshotID      string              `json:"snapshot_id"`
	WorkspaceID     string              `json:"workspace_id"`
	ProfileID       string              `json:"profile_id,omitempty"`
	Peer            string              `json:"peer"`
	Git             SnapshotGitMetadata `json:"git"`
	Counts          map[string]int      `json:"counts"`
	Entries         []SnapshotEntry     `json:"entries"`
}

type SnapshotDiff struct {
	FromSnapshotID string          `json:"from_snapshot_id"`
	ToSnapshotID   string          `json:"to_snapshot_id"`
	Added          []SnapshotEntry `json:"added"`
	Removed        []SnapshotEntry `json:"removed"`
	Changed        []SnapshotEntry `json:"changed"`
}

type SnapshotRollbackMetadata struct {
	AdapterOwned     bool   `json:"adapter_owned"`
	Applied          bool   `json:"applied"`
	FromSnapshotID   string `json:"from_snapshot_id"`
	TargetSnapshotID string `json:"target_snapshot_id"`
	Operation        string `json:"operation"`
	Note             string `json:"note"`
}

func (s *Service) ExportSnapshotManifest(ctx context.Context, params SnapshotParams) (SnapshotManifest, error) {
	workspaceID := firstNonBlank(params.WorkspaceID, s.workspaceID)
	profileID := strings.TrimSpace(params.ProfileID)
	peer := strings.TrimSpace(params.Peer)
	if workspaceID == "" || peer == "" {
		return SnapshotManifest{}, fmt.Errorf("goncho: snapshot workspace_id and peer are required")
	}
	entries := []SnapshotEntry{}
	profileEntries, err := snapshotProfileEntries(ctx, s.db, workspaceID, profileID, s.observer, peer)
	if err != nil {
		return SnapshotManifest{}, err
	}
	entries = append(entries, profileEntries...)
	conclusionEntries, err := snapshotConclusionEntries(ctx, s.db, workspaceID, profileID, s.observer, peer)
	if err != nil {
		return SnapshotManifest{}, err
	}
	entries = append(entries, conclusionEntries...)
	slotEntries, err := snapshotSlotEntries(ctx, s.db, workspaceID, profileID, peer)
	if err != nil {
		return SnapshotManifest{}, err
	}
	entries = append(entries, slotEntries...)
	actionEntries, err := snapshotActionEntries(ctx, s.db, workspaceID, profileID, peer)
	if err != nil {
		return SnapshotManifest{}, err
	}
	entries = append(entries, actionEntries...)
	sortSnapshotEntries(entries)
	manifest := SnapshotManifest{
		ManifestVersion: "goncho-snapshot-v1",
		WorkspaceID:     workspaceID,
		ProfileID:       profileID,
		Peer:            peer,
		Git: SnapshotGitMetadata{
			AdapterOwned: true,
			Operation:    "none",
			Note:         "manifest export is deterministic; git add/commit/diff/rollback are host-adapter owned",
		},
		Counts:  snapshotCounts(entries),
		Entries: entries,
	}
	manifest.SnapshotID = snapshotManifestID(manifest)
	return manifest, nil
}

func DiffSnapshotManifests(from, to SnapshotManifest) SnapshotDiff {
	fromByKey := map[string]SnapshotEntry{}
	toByKey := map[string]SnapshotEntry{}
	for _, entry := range from.Entries {
		fromByKey[entry.Key] = entry
	}
	for _, entry := range to.Entries {
		toByKey[entry.Key] = entry
	}
	diff := SnapshotDiff{FromSnapshotID: from.SnapshotID, ToSnapshotID: to.SnapshotID, Added: []SnapshotEntry{}, Removed: []SnapshotEntry{}, Changed: []SnapshotEntry{}}
	for key, entry := range toByKey {
		old, ok := fromByKey[key]
		if !ok {
			diff.Added = append(diff.Added, entry)
			continue
		}
		if old.Checksum != entry.Checksum {
			diff.Changed = append(diff.Changed, entry)
		}
	}
	for key, entry := range fromByKey {
		if _, ok := toByKey[key]; !ok {
			diff.Removed = append(diff.Removed, entry)
		}
	}
	sortSnapshotEntries(diff.Added)
	sortSnapshotEntries(diff.Removed)
	sortSnapshotEntries(diff.Changed)
	return diff
}

func BuildSnapshotRollbackMetadata(from, target SnapshotManifest) SnapshotRollbackMetadata {
	return SnapshotRollbackMetadata{
		AdapterOwned:     true,
		Applied:          false,
		FromSnapshotID:   from.SnapshotID,
		TargetSnapshotID: target.SnapshotID,
		Operation:        "rollback_metadata_only",
		Note:             "Goncho does not run git or mutate state here; host adapter owns checkout/apply/commit workflow",
	}
}

func snapshotProfileEntries(ctx context.Context, db *sql.DB, workspaceID, profileID, observer, peer string) ([]SnapshotEntry, error) {
	card, err := getPeerCard(ctx, db, workspaceID, profileID, observer, peer)
	if err != nil {
		return nil, err
	}
	if len(card) == 0 {
		return []SnapshotEntry{}, nil
	}
	return []SnapshotEntry{snapshotEntry("profile", "profile:"+workspaceID+":"+profileID+":"+peer, card, map[string]string{"profile_id": profileID, "peer": peer})}, nil
}

func snapshotConclusionEntries(ctx context.Context, db *sql.DB, workspaceID, profileID, observer, peer string) ([]SnapshotEntry, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, COALESCE(session_key, ''), content, kind, status, source, scope, updated_at
		FROM goncho_conclusions
		WHERE workspace_id = ? AND profile_id = ? AND observer_peer_id = ? AND peer_id = ?
		ORDER BY id ASC
	`, workspaceID, profileID, observer, peer)
	if err != nil {
		return nil, fmt.Errorf("goncho: snapshot conclusions: %w", err)
	}
	defer rows.Close()
	out := []SnapshotEntry{}
	for rows.Next() {
		var id, updatedAt int64
		var sessionKey, content, kind, status, source, scope string
		if err := rows.Scan(&id, &sessionKey, &content, &kind, &status, &source, &scope, &updatedAt); err != nil {
			return nil, fmt.Errorf("goncho: scan snapshot conclusion: %w", err)
		}
		out = append(out, snapshotEntry("conclusion", fmt.Sprintf("conclusion:%d", id), map[string]any{"id": id, "session_key": sessionKey, "content": content, "kind": kind, "status": status, "source": source, "scope": scope}, map[string]string{"session_key": sessionKey, "kind": kind, "status": status, "source": source, "scope": scope}))
	}
	return out, rows.Err()
}

func snapshotSlotEntries(ctx context.Context, db *sql.DB, workspaceID, profileID, peer string) ([]SnapshotEntry, error) {
	present, err := sqliteTableExists(ctx, db, "goncho_memory_slots")
	if err != nil || !present {
		return []SnapshotEntry{}, err
	}
	rows, err := db.QueryContext(ctx, `SELECT scope, name, kind, value, revision, deleted FROM goncho_memory_slots WHERE workspace_id = ? AND profile_id = ? AND peer_id = ? ORDER BY scope ASC, name ASC`, workspaceID, profileID, peer)
	if err != nil {
		return nil, fmt.Errorf("goncho: snapshot slots: %w", err)
	}
	defer rows.Close()
	out := []SnapshotEntry{}
	for rows.Next() {
		var scope, name, kind, value string
		var revision int
		var deleted bool
		if err := rows.Scan(&scope, &name, &kind, &value, &revision, dbscan.Bool(&deleted)); err != nil {
			return nil, fmt.Errorf("goncho: scan snapshot slot: %w", err)
		}
		out = append(out, snapshotEntry("slot", "slot:"+scope+":"+name, map[string]any{"scope": scope, "name": name, "kind": kind, "value": value, "revision": revision, "deleted": deleted}, map[string]string{"scope": scope, "name": name, "kind": kind}))
	}
	return out, rows.Err()
}

func snapshotActionEntries(ctx context.Context, db *sql.DB, workspaceID, profileID, peer string) ([]SnapshotEntry, error) {
	present, err := sqliteTableExists(ctx, db, "goncho_actions")
	if err != nil || !present {
		return []SnapshotEntry{}, err
	}
	nodes, err := listActionNodes(ctx, db, workspaceID, profileID, peer)
	if err != nil {
		return nil, err
	}
	out := []SnapshotEntry{}
	for _, node := range nodes {
		deps, err := listActionDependencies(ctx, db, workspaceID, profileID, peer, node.ActionID)
		if err != nil {
			return nil, err
		}
		out = append(out, snapshotEntry("action", "action:"+node.ActionID, map[string]any{"action_id": node.ActionID, "title": node.Title, "status": node.Status, "depends_on": deps}, map[string]string{"action_id": node.ActionID, "status": string(node.Status)}))
	}
	return out, nil
}

func snapshotEntry(kind, key string, payload any, metadata map[string]string) SnapshotEntry {
	return SnapshotEntry{Kind: kind, Key: key, Checksum: hashutil.JSONSHA256Hex(payload), Metadata: metadata}
}

func snapshotCounts(entries []SnapshotEntry) map[string]int {
	counts := map[string]int{}
	for _, entry := range entries {
		counts[entry.Kind]++
	}
	return counts
}

func sortSnapshotEntries(entries []SnapshotEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Kind != entries[j].Kind {
			return entries[i].Kind < entries[j].Kind
		}
		return entries[i].Key < entries[j].Key
	})
}

func snapshotManifestID(manifest SnapshotManifest) string {
	view := struct {
		ManifestVersion string          `json:"manifest_version"`
		WorkspaceID     string          `json:"workspace_id"`
		ProfileID       string          `json:"profile_id,omitempty"`
		Peer            string          `json:"peer"`
		Counts          map[string]int  `json:"counts"`
		Entries         []SnapshotEntry `json:"entries"`
	}{ManifestVersion: manifest.ManifestVersion, WorkspaceID: manifest.WorkspaceID, ProfileID: manifest.ProfileID, Peer: manifest.Peer, Counts: manifest.Counts, Entries: manifest.Entries}
	return "snap:" + hashutil.JSONSHA256HexPrefix(view, 12)
}

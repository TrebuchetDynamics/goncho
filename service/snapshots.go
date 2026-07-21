package goncho

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/dbscan"
	"github.com/TrebuchetDynamics/goncho/service/internal/hashutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/scopekey"
)

type SnapshotParams struct {
	WorkspaceID string `json:"workspace_id,omitempty"`
	ProfileID   string `json:"profile_id,omitempty"`
	Peer        string `json:"peer"`
	Branch      string `json:"branch,omitempty"`
	Worktree    string `json:"worktree,omitempty"`
	Commit      string `json:"commit,omitempty"`
}

type SnapshotGitMetadata struct {
	AdapterOwned     bool   `json:"adapter_owned"`
	Operation        string `json:"operation"`
	Note             string `json:"note"`
	Branch           string `json:"branch,omitempty"`
	Worktree         string `json:"worktree,omitempty"`
	Commit           string `json:"commit,omitempty"`
	WorktreeRedacted bool   `json:"worktree_redacted,omitempty"`
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
	CheckpointID    string              `json:"checkpoint_id,omitempty"`
	WorkspaceID     string              `json:"workspace_id"`
	ProfileID       string              `json:"profile_id,omitempty"`
	Peer            string              `json:"peer"`
	Git             SnapshotGitMetadata `json:"git"`
	Counts          map[string]int      `json:"counts"`
	Entries         []SnapshotEntry     `json:"entries"`
}

const SnapshotWarningStaleBranch = "stale_branch"

type SnapshotWarning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	From    string `json:"from,omitempty"`
	To      string `json:"to,omitempty"`
}

type SnapshotDiff struct {
	FromSnapshotID string            `json:"from_snapshot_id"`
	ToSnapshotID   string            `json:"to_snapshot_id"`
	Added          []SnapshotEntry   `json:"added"`
	Removed        []SnapshotEntry   `json:"removed"`
	Changed        []SnapshotEntry   `json:"changed"`
	GitChanged     bool              `json:"git_changed,omitempty"`
	Warnings       []SnapshotWarning `json:"warnings,omitempty"`
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
	scope := scopekey.Normalize(s.workspaceID, params.WorkspaceID, params.ProfileID, params.Peer)
	if !scope.Complete() {
		return SnapshotManifest{}, fmt.Errorf("goncho: snapshot workspace_id and peer are required")
	}
	entries := []SnapshotEntry{}
	profileEntries, err := snapshotProfileEntries(ctx, s.db, scope.WorkspaceID, scope.ProfileID, s.observer, scope.Peer)
	if err != nil {
		return SnapshotManifest{}, err
	}
	entries = append(entries, profileEntries...)
	conclusionEntries, err := snapshotConclusionEntries(ctx, s.db, scope.WorkspaceID, scope.ProfileID, s.observer, scope.Peer)
	if err != nil {
		return SnapshotManifest{}, err
	}
	entries = append(entries, conclusionEntries...)
	slotEntries, err := snapshotSlotEntries(ctx, s.db, scope.WorkspaceID, scope.ProfileID, scope.Peer)
	if err != nil {
		return SnapshotManifest{}, err
	}
	entries = append(entries, slotEntries...)
	actionEntries, err := snapshotActionEntries(ctx, s.db, scope.WorkspaceID, scope.ProfileID, scope.Peer)
	if err != nil {
		return SnapshotManifest{}, err
	}
	entries = append(entries, actionEntries...)
	sortSnapshotEntries(entries)
	manifest := SnapshotManifest{
		ManifestVersion: "goncho-snapshot-v1",
		WorkspaceID:     scope.WorkspaceID,
		ProfileID:       scope.ProfileID,
		Peer:            scope.Peer,
		Git:             snapshotGitMetadata(params),
		Counts:          snapshotCounts(entries),
		Entries:         entries,
	}
	manifest.SnapshotID = snapshotManifestID(manifest)
	if manifest.Git.Branch != "" || manifest.Git.Worktree != "" || manifest.Git.Commit != "" {
		manifest.CheckpointID = snapshotCheckpointID(manifest)
	}
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
	diff := SnapshotDiff{FromSnapshotID: from.SnapshotID, ToSnapshotID: to.SnapshotID, Added: []SnapshotEntry{}, Removed: []SnapshotEntry{}, Changed: []SnapshotEntry{}, Warnings: []SnapshotWarning{}}
	diff.GitChanged = from.Git.Branch != to.Git.Branch || from.Git.Worktree != to.Git.Worktree || from.Git.Commit != to.Git.Commit
	if from.Git.Branch != "" && to.Git.Branch != "" && from.Git.Branch != to.Git.Branch {
		diff.Warnings = append(diff.Warnings, SnapshotWarning{Code: SnapshotWarningStaleBranch, Message: "snapshot was captured on a different branch; verify remembered project state before reuse", From: from.Git.Branch, To: to.Git.Branch})
	}
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

func snapshotGitMetadata(params SnapshotParams) SnapshotGitMetadata {
	branch, worktree, commit := strings.TrimSpace(params.Branch), strings.TrimSpace(params.Worktree), strings.TrimSpace(params.Commit)
	redacted := false
	if filepath.IsAbs(worktree) {
		worktree = filepath.Base(filepath.Clean(worktree))
		redacted = true
	}
	metadata := SnapshotGitMetadata{AdapterOwned: true, Operation: "none", Note: "manifest export is deterministic; git add/commit/diff/rollback are host-adapter owned", Branch: branch, Worktree: worktree, Commit: commit, WorktreeRedacted: redacted}
	if branch != "" || worktree != "" || commit != "" {
		metadata.Operation = "capture_metadata"
		metadata.Note = "branch, worktree, and commit are host-provided evidence; Goncho does not run git"
	}
	return metadata
}

func snapshotCheckpointID(manifest SnapshotManifest) string {
	view := struct {
		SnapshotID string `json:"snapshot_id"`
		Branch     string `json:"branch,omitempty"`
		Worktree   string `json:"worktree,omitempty"`
		Commit     string `json:"commit,omitempty"`
	}{manifest.SnapshotID, manifest.Git.Branch, manifest.Git.Worktree, manifest.Git.Commit}
	return "checkpoint:" + hashutil.JSONSHA256HexPrefix(view, 12)
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

package localmarkdown

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/internal/stringutil"
	memory "github.com/TrebuchetDynamics/goncho/memory"
)

// Config wires the local-first Goncho V1 memory tools to a
// SQLite database and a Markdown export file.
type Config struct {
	Path           string
	AgentID        string
	WorkspaceID    string
	ObserverPeerID string
	PeerID         string
	SessionID      string
}

// Status is the operator-facing status for the local memory
// backend used by Memory V1 MCP tools.
type Status struct {
	Enabled         bool     `json:"enabled"`
	Path            string   `json:"path"`
	LocalFirst      bool     `json:"local_first"`
	SQLiteBacked    bool     `json:"sqlite_backed"`
	MarkdownBacked  bool     `json:"markdown_backed"`
	NetworkRequired bool     `json:"network_required"`
	OllamaRequired  bool     `json:"ollama_required"`
	MCPTools        []string `json:"mcp_tools"`
	Evidence        []string `json:"evidence,omitempty"`
}

// Entry is a single unit of agent-managed memory stored by the local Markdown
// memory backend.
type Entry struct {
	ID         string            `json:"id"`
	Content    string            `json:"content"`
	Tags       []string          `json:"tags"`
	Importance float64           `json:"importance"`
	SessionID  string            `json:"session_id,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// Store persists tool memories into the Goncho V1 SQLite
// table and mirrors the table to a human-editable Markdown file.
type Store struct {
	db  *sql.DB
	cfg Config
}

func NewStore(db *sql.DB, cfg Config) *Store {
	return &Store{db: db, cfg: cfg}
}

func (s *Store) Status(ctx context.Context) (Status, error) {
	if s == nil || s.db == nil {
		return Status{}, errors.New("goncho: nil local markdown memory store")
	}
	if strings.TrimSpace(s.cfg.Path) == "" {
		return Status{}, errors.New("goncho: local markdown memory path is required")
	}
	return Status{
		Enabled:         true,
		Path:            s.cfg.Path,
		LocalFirst:      true,
		SQLiteBacked:    true,
		MarkdownBacked:  true,
		NetworkRequired: false,
		OllamaRequired:  false,
		MCPTools:        memoryV1ToolNames(),
		Evidence:        []string{"sqlite", "markdown_export", "no_network", "ollama_optional"},
	}, nil
}

func (s *Store) Store(ctx context.Context, entry Entry) error {
	if err := s.validate(); err != nil {
		return err
	}
	entry.Content = strings.TrimSpace(entry.Content)
	if entry.Content == "" {
		return errors.New("goncho: memory content is required")
	}
	now := time.Now().UTC()
	if entry.ID == "" {
		entry.ID = fmt.Sprintf("mem_%d", now.UnixNano())
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = now
	}
	if entry.UpdatedAt.IsZero() {
		entry.UpdatedAt = now
	}
	revision, err := s.nextRevision(ctx, entry.ID)
	if err != nil {
		return err
	}
	item := memory.GonchoMemoryV1Item{
		MemoryID:       entry.ID,
		Revision:       revision,
		AgentID:        s.agentID(),
		WorkspaceID:    s.workspaceID(),
		PeerID:         s.peerID(),
		SessionID:      stringutil.FirstNonEmpty(entry.SessionID, s.sessionID()),
		Scope:          "private",
		State:          "active",
		SourceKind:     "tool",
		Checksum:       memory.GonchoMemoryV1Checksum(entry.Content),
		Tags:           append([]string(nil), entry.Tags...),
		Importance:     clampMemoryImportance(entry.Importance),
		CreatedAt:      entry.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:      entry.UpdatedAt.UTC().Format(time.RFC3339),
		ProvenanceJSON: localMarkdownProvenance(entry.Metadata),
		Content:        entry.Content,
	}
	if err := memory.ValidateGonchoMemoryV1Item(item); err != nil {
		return err
	}
	if err := s.upsertItem(ctx, item, true); err != nil {
		return err
	}
	return s.exportMarkdown(ctx)
}

func (s *Store) Retrieve(ctx context.Context, query string, limit int) ([]Entry, error) {
	if err := s.validate(); err != nil {
		return nil, err
	}
	if err := s.reloadMarkdownIfPresent(ctx); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 5
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT memory_id, content, tags_json, importance, session_key, created_at, updated_at
		FROM goncho_memory_items
		WHERE active = 1
		  AND agent_id = ?
		  AND workspace_id = ?
		  AND peer_id = ?
		ORDER BY updated_at DESC, memory_id ASC
		LIMIT 200
	`, s.agentID(), s.workspaceID(), s.peerID())
	if err != nil {
		return nil, fmt.Errorf("goncho: retrieve local markdown memory: %w", err)
	}
	defer rows.Close()

	var candidates []Entry
	for rows.Next() {
		var entry Entry
		var tagsRaw string
		var sessionKey string
		var createdAt, updatedAt int64
		if err := rows.Scan(&entry.ID, &entry.Content, &tagsRaw, &entry.Importance, &sessionKey, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("goncho: scan local markdown memory: %w", err)
		}
		_ = json.Unmarshal([]byte(tagsRaw), &entry.Tags)
		entry.SessionID = sessionKey
		entry.Metadata = map[string]string{"session_id": sessionKey}
		entry.CreatedAt = time.Unix(createdAt, 0).UTC()
		entry.UpdatedAt = time.Unix(updatedAt, 0).UTC()
		candidates = append(candidates, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("goncho: local markdown memory rows: %w", err)
	}
	query = strings.TrimSpace(query)
	ranked := rankEntriesByQuery(candidates, query, time.Now().UTC())
	capacity := limit
	if len(ranked) < capacity {
		capacity = len(ranked)
	}
	out := make([]Entry, 0, capacity)
	for _, item := range ranked {
		if query != "" && item.relevance <= 0 {
			continue
		}
		out = append(out, item.entry)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *Store) Update(ctx context.Context, id string, content string) error {
	if err := s.validate(); err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	content = strings.TrimSpace(content)
	if id == "" || content == "" {
		return errors.New("goncho: memory id and content are required")
	}
	item, found, err := s.readItem(ctx, id)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	item.Content = content
	item.Revision++
	item.State = "active"
	item.Checksum = memory.GonchoMemoryV1Checksum(content)
	item.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	item.TombstonedAt = ""
	item.TombstoneReason = ""
	if err := s.upsertItem(ctx, item, true); err != nil {
		return err
	}
	return s.exportMarkdown(ctx)
}

func (s *Store) UpdateImportance(ctx context.Context, id string, importance float64) error {
	if err := s.validate(); err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("goncho: memory id is required")
	}
	item, found, err := s.readItem(ctx, id)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	item.Importance = clampMemoryImportance(importance)
	item.Revision++
	item.State = "active"
	item.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	item.TombstonedAt = ""
	item.TombstoneReason = ""
	if err := s.upsertItem(ctx, item, true); err != nil {
		return err
	}
	return s.exportMarkdown(ctx)
}

func (s *Store) Forget(ctx context.Context, id string) error {
	if err := s.validate(); err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("goncho: memory id is required")
	}
	item, found, err := s.readItem(ctx, id)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	now := time.Now().UTC()
	item.Revision++
	item.State = "tombstoned"
	item.UpdatedAt = now.Format(time.RFC3339)
	item.TombstonedAt = now.Format(time.RFC3339)
	item.TombstoneReason = "forgotten"
	if err := s.upsertItem(ctx, item, false); err != nil {
		return err
	}
	return s.exportMarkdown(ctx)
}

func (s *Store) validate() error {
	if s == nil || s.db == nil {
		return errors.New("goncho: nil local markdown memory store")
	}
	if strings.TrimSpace(s.cfg.Path) == "" {
		return errors.New("goncho: local markdown memory path is required")
	}
	return nil
}

func (s *Store) reloadMarkdownIfPresent(ctx context.Context) error {
	if _, err := os.Stat(s.cfg.Path); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return fmt.Errorf("goncho: stat local markdown memory: %w", err)
	}
	_, err := memory.NewGonchoMarkdownStore(s.db, memory.GonchoMarkdownStoreConfig{
		Path:                  s.cfg.Path,
		DefaultObserverPeerID: s.observerPeerID(),
	}).Reload(ctx)
	return err
}

func (s *Store) exportMarkdown(ctx context.Context) error {
	_, err := memory.NewGonchoMarkdownStore(s.db, memory.GonchoMarkdownStoreConfig{
		Path:                  s.cfg.Path,
		DefaultObserverPeerID: s.observerPeerID(),
	}).Export(ctx)
	return err
}

func (s *Store) nextRevision(ctx context.Context, id string) (int, error) {
	var current int
	err := s.db.QueryRowContext(ctx, `SELECT revision FROM goncho_memory_items WHERE memory_id = ?`, id).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		return 1, nil
	}
	if err != nil {
		return 0, fmt.Errorf("goncho: read local markdown memory revision: %w", err)
	}
	return current + 1, nil
}

func (s *Store) readItem(ctx context.Context, id string) (memory.GonchoMemoryV1Item, bool, error) {
	var item memory.GonchoMemoryV1Item
	var active int
	var tombstonedAt sql.NullInt64
	var tombstoneReason sql.NullString
	var tagsRaw string
	var createdAt, updatedAt int64
	err := s.db.QueryRowContext(ctx, `
		SELECT memory_id, agent_id, workspace_id, peer_id, session_key, source_kind,
		       content, revision, active, tombstoned_at, tombstone_reason, scope,
		       provenance_json, tags_json, importance, created_at, updated_at
		FROM goncho_memory_items
		WHERE memory_id = ?
		  AND agent_id = ?
		  AND workspace_id = ?
		  AND peer_id = ?
	`, id, s.agentID(), s.workspaceID(), s.peerID()).Scan(&item.MemoryID, &item.AgentID, &item.WorkspaceID, &item.PeerID, &item.SessionID, &item.SourceKind, &item.Content, &item.Revision, &active, &tombstonedAt, &tombstoneReason, &item.Scope, &item.ProvenanceJSON, &tagsRaw, &item.Importance, &createdAt, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return memory.GonchoMemoryV1Item{}, false, nil
	}
	if err != nil {
		return memory.GonchoMemoryV1Item{}, false, fmt.Errorf("goncho: read local markdown memory item: %w", err)
	}
	item.State = "active"
	if active == 0 {
		item.State = "tombstoned"
	}
	item.CreatedAt = time.Unix(createdAt, 0).UTC().Format(time.RFC3339)
	item.UpdatedAt = time.Unix(updatedAt, 0).UTC().Format(time.RFC3339)
	if tombstonedAt.Valid {
		item.TombstonedAt = time.Unix(tombstonedAt.Int64, 0).UTC().Format(time.RFC3339)
	}
	if tombstoneReason.Valid {
		item.TombstoneReason = tombstoneReason.String
	}
	_ = json.Unmarshal([]byte(tagsRaw), &item.Tags)
	item.Checksum = memory.GonchoMemoryV1Checksum(item.Content)
	return item, true, nil
}

func (s *Store) upsertItem(ctx context.Context, item memory.GonchoMemoryV1Item, active bool) error {
	tags, err := json.Marshal(item.Tags)
	if err != nil {
		return fmt.Errorf("goncho: encode memory tags: %w", err)
	}
	createdAt, err := parseMemoryTime(item.CreatedAt)
	if err != nil {
		return err
	}
	updatedAt, err := parseMemoryTime(item.UpdatedAt)
	if err != nil {
		return err
	}
	tombstonedAt, err := parseNullableMemoryTime(item.TombstonedAt)
	if err != nil {
		return err
	}
	activeInt := 0
	if active {
		activeInt = 1
	}
	if item.ProvenanceJSON == "" {
		item.ProvenanceJSON = "{}"
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO goncho_memory_items(
			memory_id, contract_version, agent_id, workspace_id, observer_peer_id,
			peer_id, session_key, source_kind, content, revision, active,
			tombstoned_at, tombstone_reason, scope, provenance_json, tags_json,
			importance, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(memory_id) DO UPDATE SET
			contract_version = excluded.contract_version,
			agent_id = excluded.agent_id,
			workspace_id = excluded.workspace_id,
			observer_peer_id = excluded.observer_peer_id,
			peer_id = excluded.peer_id,
			session_key = excluded.session_key,
			source_kind = excluded.source_kind,
			content = excluded.content,
			revision = excluded.revision,
			active = excluded.active,
			tombstoned_at = excluded.tombstoned_at,
			tombstone_reason = excluded.tombstone_reason,
			scope = excluded.scope,
			provenance_json = excluded.provenance_json,
			tags_json = excluded.tags_json,
			importance = excluded.importance,
			updated_at = excluded.updated_at
	`, item.MemoryID, memory.GonchoMemoryV1ContractVersion, item.AgentID, item.WorkspaceID, s.observerPeerID(), item.PeerID, item.SessionID, item.SourceKind, item.Content, item.Revision, activeInt, tombstonedAt, item.TombstoneReason, item.Scope, item.ProvenanceJSON, string(tags), item.Importance, createdAt, updatedAt)
	if err != nil {
		return fmt.Errorf("goncho: upsert local markdown memory %s: %w", item.MemoryID, err)
	}
	return nil
}

func (s *Store) agentID() string {
	return stringutil.FirstNonEmpty(s.cfg.AgentID, "default-agent")
}

func (s *Store) workspaceID() string {
	return stringutil.FirstNonEmpty(s.cfg.WorkspaceID, "default-workspace")
}

func (s *Store) observerPeerID() string {
	return stringutil.FirstNonEmpty(s.cfg.ObserverPeerID, s.agentID())
}

func (s *Store) peerID() string {
	return stringutil.FirstNonEmpty(s.cfg.PeerID, s.agentID())
}

func (s *Store) sessionID() string {
	return strings.TrimSpace(s.cfg.SessionID)
}

func parseMemoryTime(value string) (int64, error) {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return 0, fmt.Errorf("goncho: parse memory timestamp %q: %w", value, err)
	}
	return parsed.Unix(), nil
}

func parseNullableMemoryTime(value string) (sql.NullInt64, error) {
	if strings.TrimSpace(value) == "" {
		return sql.NullInt64{}, nil
	}
	parsed, err := parseMemoryTime(value)
	if err != nil {
		return sql.NullInt64{}, err
	}
	return sql.NullInt64{Int64: parsed, Valid: true}, nil
}

func clampMemoryImportance(value float64) float64 {
	switch {
	case value < 0:
		return 0
	case value > 1:
		return 1
	default:
		return value
	}
}

func localMarkdownProvenance(meta map[string]string) string {
	payload := map[string]any{"source": "gormes_memory_v1_tool"}
	if len(meta) > 0 {
		payload["metadata"] = meta
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(raw)
}



func memoryV1ToolNames() []string {
	return []string{"forget_memory", "retrieve_memory", "store_memory", "summarize_memories", "update_memory"}
}

const defaultDecayHalfLife = 30 * 24 * time.Hour

type scoredEntry struct {
	entry               Entry
	relevance           float64
	recency             float64
	effectiveImportance float64
	score               float64
}

func rankEntriesByQuery(entries []Entry, query string, now time.Time) []scoredEntry {
	out := make([]scoredEntry, 0, len(entries))
	for _, entry := range entries {
		relevance := entryRelevance(entry, query)
		recency := recencyScore(memoryReferenceTime(entry), now)
		effectiveImportance := effectiveImportance(entry, now)
		out = append(out, scoredEntry{
			entry:               entry,
			relevance:           relevance,
			recency:             recency,
			effectiveImportance: effectiveImportance,
			score:               clamp01(0.3*recency + 0.5*effectiveImportance + 0.2*relevance),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].score != out[j].score {
			return out[i].score > out[j].score
		}
		if out[i].effectiveImportance != out[j].effectiveImportance {
			return out[i].effectiveImportance > out[j].effectiveImportance
		}
		iTime := memoryReferenceTime(out[i].entry)
		jTime := memoryReferenceTime(out[j].entry)
		if !iTime.Equal(jTime) {
			return iTime.After(jTime)
		}
		return out[i].entry.ID < out[j].entry.ID
	})
	return out
}

func effectiveImportance(entry Entry, now time.Time) float64 {
	base := clamp01(entry.Importance) * recencyScore(memoryReferenceTime(entry), now)
	if base < 0.01 {
		base = 0.01
	}
	return base
}

func recencyScore(createdAt time.Time, now time.Time) float64 {
	age := now.Sub(createdAt)
	if age <= 0 {
		return 1.0
	}
	halfLives := float64(age) / float64(defaultDecayHalfLife)
	return math.Exp2(-halfLives)
}

func entryRelevance(entry Entry, query string) float64 {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return 0
	}
	content := strings.ToLower(entry.Content)
	if strings.Contains(content, query) {
		return 1
	}
	tokens := strings.Fields(query)
	if len(tokens) == 0 {
		return 0
	}
	hits := 0
	for _, token := range tokens {
		if strings.Contains(content, token) {
			hits++
			continue
		}
		for _, tag := range entry.Tags {
			if strings.Contains(strings.ToLower(tag), token) {
				hits++
				break
			}
		}
	}
	return clamp01(float64(hits) / float64(len(tokens)))
}

func memoryReferenceTime(entry Entry) time.Time {
	if !entry.UpdatedAt.IsZero() {
		return entry.UpdatedAt
	}
	if !entry.CreatedAt.IsZero() {
		return entry.CreatedAt
	}
	return time.Now().UTC()
}

func clamp01(value float64) float64 {
	switch {
	case value < 0:
		return 0
	case value > 1:
		return 1
	default:
		return value
	}
}

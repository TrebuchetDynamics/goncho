package memory

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	GonchoMemoryV1ContractVersion = "1"
	GonchoMemoryV1MarkdownFormat  = "1"
	GonchoMemoryV1MCPToolContract = "1"

	gonchoMemoryV1MarkerStart     = "<!-- goncho-memory"
	gonchoMemoryV1MarkerEnd       = "-->"
	gonchoMemoryV1ClosingMarker   = "<!-- /goncho-memory -->"
	gonchoMemoryV1PrivateScope    = "private"
	gonchoMemoryV1SharedScope     = "shared"
	gonchoMemoryV1StateActive     = "active"
	gonchoMemoryV1StateTombstoned = "tombstoned"
)

type GonchoMemoryV1Document struct {
	FormatVersion   string               `json:"format_version"`
	ContractVersion string               `json:"contract_version"`
	Items           []GonchoMemoryV1Item `json:"items"`
}

type GonchoMemoryV1Item struct {
	MemoryID        string   `json:"memory_id" yaml:"memory_id"`
	Revision        int      `json:"revision" yaml:"revision"`
	AgentID         string   `json:"agent_id" yaml:"agent_id"`
	WorkspaceID     string   `json:"workspace_id" yaml:"workspace_id"`
	PeerID          string   `json:"peer_id" yaml:"peer_id"`
	SessionID       string   `json:"session_id" yaml:"session_id"`
	Scope           string   `json:"scope" yaml:"scope"`
	State           string   `json:"state" yaml:"state"`
	SourceKind      string   `json:"source_kind" yaml:"source_kind"`
	SourceTurnID    string   `json:"source_turn_id,omitempty" yaml:"source_turn_id,omitempty"`
	TombstonedAt    string   `json:"tombstoned_at,omitempty" yaml:"tombstoned_at,omitempty"`
	TombstoneReason string   `json:"tombstone_reason,omitempty" yaml:"tombstone_reason,omitempty"`
	Checksum        string   `json:"checksum" yaml:"checksum"`
	Tags            []string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Importance      float64  `json:"importance" yaml:"importance"`
	CreatedAt       string   `json:"created_at" yaml:"created_at"`
	UpdatedAt       string   `json:"updated_at" yaml:"updated_at"`
	ProvenanceJSON  string   `json:"provenance_json,omitempty" yaml:"provenance_json,omitempty"`
	Content         string   `json:"content" yaml:"-"`
}

type GonchoMemoryV1RecallRequest struct {
	AgentID     string
	WorkspaceID string
	AllowShared bool
}

func GonchoMemoryV1Checksum(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func ValidateGonchoMemoryV1Item(item GonchoMemoryV1Item) error {
	switch {
	case strings.TrimSpace(item.MemoryID) == "":
		return errors.New("memory: memory_id is required")
	case strings.TrimSpace(item.AgentID) == "":
		return errors.New("memory: agent_id is required")
	case strings.TrimSpace(item.WorkspaceID) == "":
		return errors.New("memory: workspace_id is required")
	case strings.TrimSpace(item.PeerID) == "":
		return errors.New("memory: peer_id is required")
	case item.Revision <= 0:
		return errors.New("memory: revision must be positive")
	case item.Scope != gonchoMemoryV1PrivateScope && item.Scope != gonchoMemoryV1SharedScope:
		return fmt.Errorf("memory: unsupported scope %q", item.Scope)
	case item.State != gonchoMemoryV1StateActive && item.State != gonchoMemoryV1StateTombstoned:
		return fmt.Errorf("memory: unsupported state %q", item.State)
	case item.State == gonchoMemoryV1StateTombstoned && strings.TrimSpace(item.TombstonedAt) == "":
		return errors.New("memory: tombstoned memories require tombstoned_at")
	case strings.TrimSpace(item.Content) == "":
		return errors.New("memory: content is required")
	}
	if item.Checksum != "" && item.Checksum != GonchoMemoryV1Checksum(item.Content) {
		return fmt.Errorf("memory: checksum mismatch for %s", item.MemoryID)
	}
	return nil
}

func CanRecallGonchoMemoryV1(req GonchoMemoryV1RecallRequest, item GonchoMemoryV1Item) (bool, string) {
	if item.State == gonchoMemoryV1StateTombstoned {
		return false, "tombstoned"
	}
	if item.AgentID == req.AgentID && item.WorkspaceID == req.WorkspaceID {
		return true, "owner_agent"
	}
	if req.AllowShared && item.Scope == gonchoMemoryV1SharedScope && item.WorkspaceID == req.WorkspaceID {
		return true, "shared_workspace"
	}
	return false, "private_agent_boundary"
}

func ParseGonchoMemoryV1Markdown(body []byte) (GonchoMemoryV1Document, error) {
	text := string(body)
	header, rest, err := parseGonchoMemoryV1FrontMatter(text)
	if err != nil {
		return GonchoMemoryV1Document{}, err
	}
	doc := GonchoMemoryV1Document{
		FormatVersion:   header.FormatVersion,
		ContractVersion: header.ContractVersion,
	}
	for {
		start := strings.Index(rest, gonchoMemoryV1MarkerStart)
		if start < 0 {
			break
		}
		afterStart := rest[start+len(gonchoMemoryV1MarkerStart):]
		metaEnd := strings.Index(afterStart, gonchoMemoryV1MarkerEnd)
		if metaEnd < 0 {
			return GonchoMemoryV1Document{}, errors.New("memory: unterminated goncho-memory metadata block")
		}
		metaRaw := strings.TrimSpace(afterStart[:metaEnd])
		afterMeta := afterStart[metaEnd+len(gonchoMemoryV1MarkerEnd):]
		contentEnd := strings.Index(afterMeta, gonchoMemoryV1ClosingMarker)
		if contentEnd < 0 {
			return GonchoMemoryV1Document{}, errors.New("memory: unterminated goncho-memory content block")
		}
		var item GonchoMemoryV1Item
		if err := yaml.Unmarshal([]byte(metaRaw), &item); err != nil {
			return GonchoMemoryV1Document{}, fmt.Errorf("memory: parse goncho-memory metadata: %w", err)
		}
		item.Content = strings.Trim(afterMeta[:contentEnd], "\n")
		doc.Items = append(doc.Items, item)
		rest = afterMeta[contentEnd+len(gonchoMemoryV1ClosingMarker):]
	}
	return doc, nil
}

func RenderGonchoMemoryV1Markdown(doc GonchoMemoryV1Document) (string, error) {
	if doc.FormatVersion == "" {
		doc.FormatVersion = GonchoMemoryV1MarkdownFormat
	}
	if doc.ContractVersion == "" {
		doc.ContractVersion = GonchoMemoryV1ContractVersion
	}
	items := append([]GonchoMemoryV1Item(nil), doc.Items...)
	sort.SliceStable(items, func(i, j int) bool { return items[i].MemoryID < items[j].MemoryID })
	var b strings.Builder
	fmt.Fprintf(&b, "---\ngoncho_memory_format: %q\ngoncho_memory_contract: %q\n---\n\n", doc.FormatVersion, doc.ContractVersion)
	b.WriteString("# Goncho Memory V1 Export\n\n")
	for _, item := range items {
		if item.Checksum == "" {
			item.Checksum = GonchoMemoryV1Checksum(item.Content)
		}
		meta := item
		meta.Content = ""
		raw, err := yaml.Marshal(meta)
		if err != nil {
			return "", fmt.Errorf("memory: render goncho-memory metadata: %w", err)
		}
		b.WriteString(gonchoMemoryV1MarkerStart)
		b.WriteByte('\n')
		b.Write(raw)
		b.WriteString(gonchoMemoryV1MarkerEnd)
		b.WriteByte('\n')
		b.WriteString(strings.Trim(item.Content, "\n"))
		b.WriteByte('\n')
		b.WriteString(gonchoMemoryV1ClosingMarker)
		b.WriteString("\n\n")
	}
	return b.String(), nil
}

type gonchoMemoryV1FrontMatter struct {
	FormatVersion   string `yaml:"goncho_memory_format"`
	ContractVersion string `yaml:"goncho_memory_contract"`
}

func parseGonchoMemoryV1FrontMatter(text string) (gonchoMemoryV1FrontMatter, string, error) {
	if !strings.HasPrefix(text, "---\n") {
		return gonchoMemoryV1FrontMatter{}, "", errors.New("memory: goncho memory markdown missing frontmatter")
	}
	rest := text[len("---\n"):]
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		return gonchoMemoryV1FrontMatter{}, "", errors.New("memory: goncho memory markdown unterminated frontmatter")
	}
	var fm gonchoMemoryV1FrontMatter
	if err := yaml.Unmarshal([]byte(rest[:end]), &fm); err != nil {
		return gonchoMemoryV1FrontMatter{}, "", fmt.Errorf("memory: parse goncho memory frontmatter: %w", err)
	}
	if fm.FormatVersion == "" || fm.ContractVersion == "" {
		return gonchoMemoryV1FrontMatter{}, "", errors.New("memory: goncho memory frontmatter missing format or contract version")
	}
	return fm, rest[end+len("\n---\n"):], nil
}

type GonchoMarkdownStoreConfig struct {
	Path                  string
	DefaultObserverPeerID string
}

type GonchoMarkdownStore struct {
	db     *sql.DB
	Config GonchoMarkdownStoreConfig
}

type GonchoMarkdownReloadResult struct {
	Inserted        int
	Updated         int
	Tombstoned      int
	Conflicts       []GonchoMarkdownConflict
	NetworkRequired bool
	OllamaRequired  bool
}

type GonchoMarkdownExportResult struct {
	Exported        int
	NetworkRequired bool
	OllamaRequired  bool
}

type GonchoMarkdownConflict struct {
	MemoryID string
	Reason   string
}

func NewGonchoMarkdownStore(db *sql.DB, cfg GonchoMarkdownStoreConfig) *GonchoMarkdownStore {
	return &GonchoMarkdownStore{db: db, Config: cfg}
}

func (s *GonchoMarkdownStore) Reload(ctx context.Context) (GonchoMarkdownReloadResult, error) {
	var result GonchoMarkdownReloadResult
	if s == nil || s.db == nil {
		return result, errors.New("memory: nil goncho markdown store")
	}
	body, err := os.ReadFile(s.Config.Path)
	if err != nil {
		return result, fmt.Errorf("memory: read goncho markdown: %w", err)
	}
	doc, err := ParseGonchoMemoryV1Markdown(body)
	if err != nil {
		return result, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("memory: begin goncho markdown reload: %w", err)
	}
	defer tx.Rollback()
	for _, item := range doc.Items {
		item.Checksum = GonchoMemoryV1Checksum(item.Content)
		if err := ValidateGonchoMemoryV1Item(item); err != nil {
			return result, err
		}
		if err := upsertMarkdownItem(ctx, tx, s.Config.DefaultObserverPeerID, item); err != nil {
			return result, err
		}
		result.Inserted++
		if item.State == gonchoMemoryV1StateTombstoned {
			result.Tombstoned++
		}
	}
	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("memory: commit goncho markdown reload: %w", err)
	}
	return result, nil
}

func (s *GonchoMarkdownStore) Export(ctx context.Context) (GonchoMarkdownExportResult, error) {
	var result GonchoMarkdownExportResult
	if s == nil || s.db == nil {
		return result, errors.New("memory: nil goncho markdown store")
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT memory_id, revision, agent_id, workspace_id, peer_id, session_key, source_kind,
		       content, active, tombstoned_at, tombstone_reason, scope, provenance_json,
		       tags_json, importance, created_at, updated_at
		FROM goncho_memory_items
		ORDER BY memory_id
	`)
	if err != nil {
		return result, fmt.Errorf("memory: list goncho markdown items: %w", err)
	}
	defer rows.Close()
	doc := GonchoMemoryV1Document{FormatVersion: GonchoMemoryV1MarkdownFormat, ContractVersion: GonchoMemoryV1ContractVersion}
	for rows.Next() {
		var item GonchoMemoryV1Item
		var active int
		var tombstonedAt sql.NullInt64
		var tagsRaw string
		var createdAt, updatedAt int64
		if err := rows.Scan(&item.MemoryID, &item.Revision, &item.AgentID, &item.WorkspaceID, &item.PeerID, &item.SessionID, &item.SourceKind, &item.Content, &active, &tombstonedAt, &item.TombstoneReason, &item.Scope, &item.ProvenanceJSON, &tagsRaw, &item.Importance, &createdAt, &updatedAt); err != nil {
			return result, fmt.Errorf("memory: scan goncho markdown item: %w", err)
		}
		item.State = gonchoMemoryV1StateActive
		if active == 0 {
			item.State = gonchoMemoryV1StateTombstoned
		}
		if tombstonedAt.Valid {
			item.TombstonedAt = time.Unix(tombstonedAt.Int64, 0).UTC().Format(time.RFC3339)
		}
		_ = json.Unmarshal([]byte(tagsRaw), &item.Tags)
		item.CreatedAt = time.Unix(createdAt, 0).UTC().Format(time.RFC3339)
		item.UpdatedAt = time.Unix(updatedAt, 0).UTC().Format(time.RFC3339)
		item.Checksum = GonchoMemoryV1Checksum(item.Content)
		doc.Items = append(doc.Items, item)
	}
	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("memory: iterate goncho markdown items: %w", err)
	}
	rendered, err := RenderGonchoMemoryV1Markdown(doc)
	if err != nil {
		return result, err
	}
	if err := os.WriteFile(s.Config.Path, []byte(rendered), 0o600); err != nil {
		return result, fmt.Errorf("memory: write goncho markdown: %w", err)
	}
	result.Exported = len(doc.Items)
	return result, nil
}

func upsertMarkdownItem(ctx context.Context, tx *sql.Tx, defaultObserver string, item GonchoMemoryV1Item) error {
	tags, err := json.Marshal(item.Tags)
	if err != nil {
		return fmt.Errorf("memory: encode goncho tags: %w", err)
	}
	createdAt, err := parseMemoryTime(item.CreatedAt)
	if err != nil {
		return err
	}
	updatedAt, err := parseMemoryTime(item.UpdatedAt)
	if err != nil {
		return err
	}
	var tombstonedAt any
	if strings.TrimSpace(item.TombstonedAt) != "" {
		parsed, err := parseMemoryTime(item.TombstonedAt)
		if err != nil {
			return err
		}
		tombstonedAt = parsed
	}
	active := 1
	if item.State == gonchoMemoryV1StateTombstoned {
		active = 0
	}
	if item.ProvenanceJSON == "" {
		item.ProvenanceJSON = "{}"
	}
	observer := defaultObserver
	if observer == "" {
		observer = item.AgentID
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO goncho_memory_items(
			memory_id, contract_version, agent_id, workspace_id, observer_peer_id, peer_id,
			session_key, source_kind, content, revision, active, tombstoned_at,
			tombstone_reason, scope, provenance_json, tags_json, importance, created_at, updated_at
		)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
			created_at = excluded.created_at,
			updated_at = excluded.updated_at
	`, item.MemoryID, GonchoMemoryV1ContractVersion, item.AgentID, item.WorkspaceID, observer, item.PeerID, item.SessionID, item.SourceKind, item.Content, item.Revision, active, tombstonedAt, item.TombstoneReason, item.Scope, item.ProvenanceJSON, string(tags), item.Importance, createdAt, updatedAt)
	if err != nil {
		return fmt.Errorf("memory: upsert goncho markdown item: %w", err)
	}
	return nil
}

func parseMemoryTime(value string) (int64, error) {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return 0, fmt.Errorf("memory: parse time %q: %w", value, err)
	}
	return parsed.Unix(), nil
}

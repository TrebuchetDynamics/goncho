package goncho

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	GonchoMemoryV1ContractVersion = "1"
	GonchoMemoryV1MarkdownFormat  = "1"
	GonchoMemoryV1MCPToolContract = "1"
	gonchoMemoryV1MarkerStart     = "<!-- goncho-memory"
	gonchoMemoryV1MarkerEnd       = "-->"
	gonchoMemoryV1ClosingMarker   = "<!-- /goncho-memory -->"
	gonchoMemoryV1StateActive     = "active"
	gonchoMemoryV1StateTombstoned = "tombstoned"
)

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

func GonchoMemoryV1Checksum(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func ValidateGonchoMemoryV1Item(item GonchoMemoryV1Item) error {
	if strings.TrimSpace(item.MemoryID) == "" {
		return fmt.Errorf("goncho: memory_id is required")
	}
	if strings.TrimSpace(item.AgentID) == "" {
		return fmt.Errorf("goncho: agent_id is required")
	}
	if strings.TrimSpace(item.WorkspaceID) == "" {
		return fmt.Errorf("goncho: workspace_id is required")
	}
	if strings.TrimSpace(item.PeerID) == "" {
		return fmt.Errorf("goncho: peer_id is required")
	}
	if strings.TrimSpace(item.Content) == "" {
		return fmt.Errorf("goncho: content is required")
	}
	if item.Checksum != GonchoMemoryV1Checksum(item.Content) {
		return fmt.Errorf("goncho: checksum mismatch")
	}
	return nil
}

type GonchoMarkdownStoreConfig struct {
	Path                string
	DefaultObserverPeerID string
	AgentID             string
	WorkspaceID         string
	FilePath            string
}

type GonchoMarkdownStore struct {
	db       interface{}
	cfg      GonchoMarkdownStoreConfig
}

func (s *GonchoMarkdownStore) Reload(ctx interface{}) error {
	return nil
}

func (s *GonchoMarkdownStore) Export(ctx interface{}) error {
	if s == nil || s.cfg.Path == "" {
		return nil
	}
	db, ok := s.db.(*sql.DB)
	if !ok || db == nil {
		return nil
	}
	c, ok := ctx.(context.Context)
	if !ok {
		c = context.Background()
	}

	rows, err := db.QueryContext(c, `
		SELECT memory_id, agent_id, workspace_id, peer_id, session_key,
		       source_kind, content, revision, active, scope,
		       provenance_json, tags_json, importance, created_at, updated_at
		FROM goncho_memory_items
		WHERE active = 1
		ORDER BY updated_at DESC, memory_id ASC
	`)
	if err != nil {
		return fmt.Errorf("goncho: export markdown query: %w", err)
	}
	defer rows.Close()

	var items []GonchoMemoryV1Item
	for rows.Next() {
		var item GonchoMemoryV1Item
		var tagsRaw, provenanceRaw string
		var createdAt, updatedAt int64
		var sessionKey, sourceKind sql.NullString
		var active int
		if err := rows.Scan(&item.MemoryID, &item.AgentID, &item.WorkspaceID, &item.PeerID,
			&sessionKey, &sourceKind, &item.Content, &item.Revision, &active,
			&item.Scope, &provenanceRaw, &tagsRaw, &item.Importance, &createdAt, &updatedAt); err != nil {
			return fmt.Errorf("goncho: export markdown scan: %w", err)
		}
		item.State = "active"
		if active == 0 {
			item.State = "tombstoned"
		}
		if sessionKey.Valid {
			item.SessionID = sessionKey.String
		}
		if sourceKind.Valid {
			item.SourceKind = sourceKind.String
		}
		item.CreatedAt = time.Unix(createdAt, 0).UTC().Format(time.RFC3339)
		item.UpdatedAt = time.Unix(updatedAt, 0).UTC().Format(time.RFC3339)
		item.Tags = []string{}
		_ = json.Unmarshal([]byte(tagsRaw), &item.Tags)
		item.ProvenanceJSON = provenanceRaw
		item.Checksum = GonchoMemoryV1Checksum(item.Content)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("goncho: export markdown rows: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("<!-- goncho-memory -->\n")
	for _, it := range items {
		sb.WriteString(fmt.Sprintf("- memory_id: %s\n", it.MemoryID))
		sb.WriteString(fmt.Sprintf("  agent_id: %s\n", it.AgentID))
		sb.WriteString(fmt.Sprintf("  workspace_id: %s\n", it.WorkspaceID))
		sb.WriteString(fmt.Sprintf("  peer_id: %s\n", it.PeerID))
		if it.SessionID != "" {
			sb.WriteString(fmt.Sprintf("  session_id: %s\n", it.SessionID))
		}
		sb.WriteString(fmt.Sprintf("  importance: %.2f\n", it.Importance))
		sb.WriteString(fmt.Sprintf("  tags: %s\n", strings.Join(it.Tags, ", ")))
		sb.WriteString(fmt.Sprintf("  created_at: %s\n", it.CreatedAt))
		sb.WriteString(fmt.Sprintf("  updated_at: %s\n", it.UpdatedAt))
		sb.WriteString(fmt.Sprintf("  content: |\n"))
		for _, line := range strings.Split(it.Content, "\n") {
			sb.WriteString(fmt.Sprintf("    %s\n", line))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("<!-- /goncho-memory -->\n")

	if err := os.MkdirAll(filepath.Dir(s.cfg.Path), 0o755); err != nil {
		return fmt.Errorf("goncho: export markdown mkdir: %w", err)
	}
	return os.WriteFile(s.cfg.Path, []byte(sb.String()), 0o644)
}

func NewGonchoMarkdownStore(db interface{}, cfg GonchoMarkdownStoreConfig) (*GonchoMarkdownStore, error) {
	return &GonchoMarkdownStore{db: db, cfg: cfg}, nil
}

// GonchoMemoryV1RecallRequest is the input for V1 memory recall authorization.
type GonchoMemoryV1RecallRequest struct {
	AgentID     string
	WorkspaceID string
	PeerID      string
	SessionID   string
	Scope       string
}

// GonchoMemoryV1Document is a parsed V1 markdown memory document.
type GonchoMemoryV1Document struct {
	FormatVersion string
	Items         []GonchoMemoryV1Item
}

// ParseGonchoMemoryV1Markdown parses a V1 markdown memory document.
func ParseGonchoMemoryV1Markdown(data []byte) (*GonchoMemoryV1Document, error) {
	content := string(data)
	doc := &GonchoMemoryV1Document{FormatVersion: GonchoMemoryV1MarkdownFormat}
	start := strings.Index(content, gonchoMemoryV1MarkerStart)
	if start < 0 {
		return doc, nil
	}
	end := strings.Index(content[start:], gonchoMemoryV1ClosingMarker)
	if end < 0 {
		return doc, nil
	}
	block := content[start : start+end+len(gonchoMemoryV1ClosingMarker)]
	lines := strings.Split(block, "\n")
	var currentItem *GonchoMemoryV1Item
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- id:") || strings.HasPrefix(line, "- memory_id:") {
			if currentItem != nil {
				doc.Items = append(doc.Items, *currentItem)
			}
			currentItem = &GonchoMemoryV1Item{MemoryID: strings.TrimSpace(strings.TrimPrefix(line, "- id:")), State: gonchoMemoryV1StateActive}
			if id := strings.TrimSpace(strings.TrimPrefix(line, "- memory_id:")); id != "" {
				currentItem.MemoryID = id
			}
		}
		if currentItem != nil {
			switch {
			case strings.HasPrefix(line, "- agent_id:"):
				currentItem.AgentID = strings.TrimSpace(strings.TrimPrefix(line, "- agent_id:"))
			case strings.HasPrefix(line, "- workspace_id:"):
				currentItem.WorkspaceID = strings.TrimSpace(strings.TrimPrefix(line, "- workspace_id:"))
			case strings.HasPrefix(line, "- peer_id:"):
				currentItem.PeerID = strings.TrimSpace(strings.TrimPrefix(line, "- peer_id:"))
			case strings.HasPrefix(line, "- scope:"):
				currentItem.Scope = strings.TrimSpace(strings.TrimPrefix(line, "- scope:"))
			case strings.HasPrefix(line, "- content:"):
				currentItem.Content = strings.TrimSpace(strings.TrimPrefix(line, "- content:"))
			}
		}
	}
	if currentItem != nil {
		doc.Items = append(doc.Items, *currentItem)
	}
	return doc, nil
}

// CanRecallGonchoMemoryV1 checks whether a recall request can access a given memory item.
func CanRecallGonchoMemoryV1(req GonchoMemoryV1RecallRequest, item GonchoMemoryV1Item) (bool, string) {
	if item.Scope == "private" && item.AgentID != req.AgentID {
		return false, "private_agent_boundary"
	}
	if req.WorkspaceID != "" && item.WorkspaceID != "" && item.WorkspaceID != req.WorkspaceID {
		return false, "workspace_mismatch"
	}
	return true, ""
}

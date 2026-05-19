package goncho

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
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
	return nil
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

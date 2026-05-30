package goncho

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/TrebuchetDynamics/goncho/service/internal/hashutil"
	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
)

// TextEmbeddingProvider turns text into deterministic local vectors. It is kept
// separate from VectorStore so embedding generation and vector storage can be
// tested, swapped, and audited independently.
type TextEmbeddingProvider interface {
	EmbedText(ctx context.Context, text string) ([]float64, error)
}

type LocalVectorIndexOptions struct {
	Path     string
	Provider TextEmbeddingProvider
}

type LocalVectorMemory struct {
	MemoryID    string            `json:"memory_id"`
	WorkspaceID string            `json:"workspace_id"`
	ProfileID   string            `json:"profile_id,omitempty"`
	Peer        string            `json:"peer"`
	SourceType  string            `json:"source_type,omitempty"`
	Content     string            `json:"content"`
	SessionID   string            `json:"session_id,omitempty"`
	AgentID     string            `json:"agent_id,omitempty"`
	ScopeID     string            `json:"scope_id,omitempty"`
	CreatedAt   time.Time         `json:"created_at,omitempty"`
	Importance  float64           `json:"importance,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type LocalVectorIndexDiagnostics struct {
	Path          string    `json:"path"`
	Dimensions    int       `json:"dimensions"`
	Count         int       `json:"count"`
	Checksum      string    `json:"checksum"`
	StaleRows     int       `json:"stale_rows"`
	LastIndexedAt time.Time `json:"last_indexed_at,omitempty"`
}

type LocalVectorIndex struct {
	mu         sync.RWMutex
	path       string
	provider   TextEmbeddingProvider
	dimensions int
	entries    []localVectorEntry
}

type localVectorEntry struct {
	LocalVectorMemory
	Vector          []float64 `json:"vector"`
	ContentChecksum string    `json:"content_checksum"`
	IndexedAt       time.Time `json:"indexed_at"`
}

type localVectorIndexFile struct {
	Version    string             `json:"version"`
	Dimensions int                `json:"dimensions"`
	Entries    []localVectorEntry `json:"entries"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

func NewLocalVectorIndex(ctx context.Context, opts LocalVectorIndexOptions) (*LocalVectorIndex, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	path := strings.TrimSpace(opts.Path)
	if path == "" {
		return nil, errors.New("goncho: local vector index path is required")
	}
	if opts.Provider == nil {
		return nil, errors.New("goncho: local vector index embedding provider is required")
	}
	idx := &LocalVectorIndex{path: path, provider: opts.Provider}
	if err := idx.load(ctx); err != nil {
		return nil, err
	}
	return idx, nil
}

func (i *LocalVectorIndex) Upsert(ctx context.Context, memory LocalVectorMemory) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if i == nil || i.provider == nil {
		return errors.New("goncho: local vector index is not initialized")
	}
	memory = normalizeLocalVectorMemory(memory)
	if memory.WorkspaceID == "" || memory.Peer == "" || memory.MemoryID == "" || memory.Content == "" {
		return errors.New("goncho: local vector memory workspace_id, peer, memory_id, and content are required")
	}
	vector, err := i.provider.EmbedText(ctx, memory.Content)
	if err != nil {
		return fmt.Errorf("goncho: embed local vector memory: %w", err)
	}
	if err := validateLocalVector(vector); err != nil {
		return err
	}
	entry := localVectorEntry{LocalVectorMemory: memory, Vector: sliceutil.Clone(vector), ContentChecksum: localVectorContentChecksum(memory.Content), IndexedAt: time.Now().UTC()}
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.dimensions == 0 {
		i.dimensions = len(vector)
	}
	if len(vector) != i.dimensions {
		return fmt.Errorf("goncho: local vector dimensions = %d, want %d", len(vector), i.dimensions)
	}
	replaced := false
	for idx := range i.entries {
		if i.entries[idx].WorkspaceID == entry.WorkspaceID && i.entries[idx].MemoryID == entry.MemoryID {
			i.entries[idx] = entry
			replaced = true
			break
		}
	}
	if !replaced {
		i.entries = append(i.entries, entry)
	}
	sortLocalVectorEntries(i.entries)
	return i.saveLocked(ctx)
}

func (i *LocalVectorIndex) Search(ctx context.Context, query VectorSearchQuery) ([]VectorSearchHit, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if i == nil || i.provider == nil {
		return nil, errors.New("goncho: local vector index is not initialized")
	}
	if strings.TrimSpace(query.Query) == "" {
		return []VectorSearchHit{}, nil
	}
	qv, err := i.provider.EmbedText(ctx, query.Query)
	if err != nil {
		return nil, fmt.Errorf("goncho: embed local vector query: %w", err)
	}
	if err := validateLocalVector(qv); err != nil {
		return nil, err
	}
	i.mu.RLock()
	defer i.mu.RUnlock()
	if i.dimensions > 0 && len(qv) != i.dimensions {
		return nil, fmt.Errorf("goncho: local vector query dimensions = %d, want %d", len(qv), i.dimensions)
	}
	type scoredHit struct {
		hit   VectorSearchHit
		score float64
	}
	scored := []scoredHit{}
	for _, entry := range i.entries {
		if !localVectorEntryMatches(query, entry) || len(entry.Vector) != len(qv) {
			continue
		}
		score := cosineSimilarity(qv, entry.Vector)
		if score <= 0 {
			continue
		}
		scored = append(scored, scoredHit{score: score, hit: VectorSearchHit{
			MemoryID:   entry.MemoryID,
			SourceType: entry.SourceType,
			Content:    entry.Content,
			SessionID:  entry.SessionID,
			AgentID:    entry.AgentID,
			ScopeID:    entry.ScopeID,
			CreatedAt:  entry.CreatedAt,
			Importance: entry.Importance,
			Score:      score,
			Metadata:   cloneVectorMetadata(entry.Metadata),
		}})
	}
	sort.SliceStable(scored, func(a, b int) bool {
		if scored[a].score != scored[b].score {
			return scored[a].score > scored[b].score
		}
		return scored[a].hit.MemoryID < scored[b].hit.MemoryID
	})
	limit := query.Limit
	if limit <= 0 || limit > len(scored) {
		limit = len(scored)
	}
	out := make([]VectorSearchHit, 0, limit)
	for _, item := range scored[:limit] {
		item.hit.Score = roundRecallFloat(item.hit.Score)
		out = append(out, item.hit)
	}
	return out, nil
}

func (i *LocalVectorIndex) Diagnostics(ctx context.Context) (LocalVectorIndexDiagnostics, error) {
	if err := ctx.Err(); err != nil {
		return LocalVectorIndexDiagnostics{}, err
	}
	if i == nil {
		return LocalVectorIndexDiagnostics{}, errors.New("goncho: local vector index is not initialized")
	}
	i.mu.RLock()
	defer i.mu.RUnlock()
	diag := LocalVectorIndexDiagnostics{Path: i.path, Dimensions: i.dimensions, Count: len(i.entries), Checksum: localVectorIndexChecksum(i.dimensions, i.entries)}
	for _, entry := range i.entries {
		if len(entry.Vector) != i.dimensions || entry.ContentChecksum != localVectorContentChecksum(entry.Content) {
			diag.StaleRows++
		}
		if entry.IndexedAt.After(diag.LastIndexedAt) {
			diag.LastIndexedAt = entry.IndexedAt
		}
	}
	return diag, nil
}

func (i *LocalVectorIndex) load(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	raw, err := os.ReadFile(i.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("goncho: read local vector index: %w", err)
	}
	var file localVectorIndexFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return fmt.Errorf("goncho: decode local vector index: %w", err)
	}
	if file.Version != "goncho-local-vector-index-v1" {
		return fmt.Errorf("goncho: unsupported local vector index version %q", file.Version)
	}
	i.dimensions = file.Dimensions
	i.entries = sliceutil.Clone(file.Entries)
	sortLocalVectorEntries(i.entries)
	return nil
}

func (i *LocalVectorIndex) saveLocked(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	file := localVectorIndexFile{Version: "goncho-local-vector-index-v1", Dimensions: i.dimensions, Entries: i.entries, UpdatedAt: time.Now().UTC()}
	raw, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("goncho: encode local vector index: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(i.path), 0o755); err != nil {
		return fmt.Errorf("goncho: create local vector index dir: %w", err)
	}
	tmp := i.path + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o600); err != nil {
		return fmt.Errorf("goncho: write local vector index temp: %w", err)
	}
	if err := os.Rename(tmp, i.path); err != nil {
		return fmt.Errorf("goncho: replace local vector index: %w", err)
	}
	return nil
}

func normalizeLocalVectorMemory(memory LocalVectorMemory) LocalVectorMemory {
	memory.MemoryID = strings.TrimSpace(memory.MemoryID)
	memory.WorkspaceID = strings.TrimSpace(memory.WorkspaceID)
	memory.ProfileID = strings.TrimSpace(memory.ProfileID)
	memory.Peer = strings.TrimSpace(memory.Peer)
	memory.SourceType = strings.TrimSpace(memory.SourceType)
	if memory.SourceType == "" {
		memory.SourceType = "vector"
	}
	memory.Content = strings.TrimSpace(memory.Content)
	memory.SessionID = strings.TrimSpace(memory.SessionID)
	memory.AgentID = strings.TrimSpace(memory.AgentID)
	memory.ScopeID = strings.TrimSpace(memory.ScopeID)
	memory.Metadata = cloneVectorMetadata(memory.Metadata)
	return memory
}

func localVectorEntryMatches(query VectorSearchQuery, entry localVectorEntry) bool {
	if strings.TrimSpace(query.WorkspaceID) != "" && entry.WorkspaceID != strings.TrimSpace(query.WorkspaceID) {
		return false
	}
	if strings.TrimSpace(query.ProfileID) != "" && entry.ProfileID != strings.TrimSpace(query.ProfileID) {
		return false
	}
	if strings.TrimSpace(query.Peer) != "" && entry.Peer != strings.TrimSpace(query.Peer) {
		return false
	}
	if strings.TrimSpace(query.SessionKey) != "" && entry.SessionID != "" && entry.SessionID != strings.TrimSpace(query.SessionKey) {
		return false
	}
	if strings.TrimSpace(query.ScopeID) != "" && entry.ScopeID != "" && entry.ScopeID != strings.TrimSpace(query.ScopeID) {
		return false
	}
	return vectorSourceAllowed(query.Sources, entry.SourceType)
}

func validateLocalVector(vector []float64) error {
	if len(vector) == 0 {
		return errors.New("goncho: embedding vector is empty")
	}
	for _, value := range vector {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return errors.New("goncho: embedding vector contains non-finite value")
		}
	}
	return nil
}

func cosineSimilarity(a, b []float64) float64 {
	var dot, normA, normB float64
	for idx := range a {
		dot += a[idx] * b[idx]
		normA += a[idx] * a[idx]
		normB += b[idx] * b[idx]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func localVectorContentChecksum(content string) string {
	return hashutil.SHA256HexString(content)
}

func localVectorIndexChecksum(dimensions int, entries []localVectorEntry) string {
	view := struct {
		Dimensions int                `json:"dimensions"`
		Entries    []localVectorEntry `json:"entries"`
	}{Dimensions: dimensions, Entries: entries}
	return hashutil.JSONSHA256Hex(view)
}

func sortLocalVectorEntries(entries []localVectorEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].WorkspaceID != entries[j].WorkspaceID {
			return entries[i].WorkspaceID < entries[j].WorkspaceID
		}
		return entries[i].MemoryID < entries[j].MemoryID
	})
}

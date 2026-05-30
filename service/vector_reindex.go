package goncho

import (
	"context"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/hashutil"
)

// ReindexPreviewResult is the non-mutating preview for embedding reindex.
type ReindexPreviewResult struct {
	Status      string `json:"status"`
	Mutates     bool   `json:"mutates"`
	Total       int    `json:"total"`        // total non-deleted conclusions
	NotIndexed  int    `json:"not_indexed"`  // conclusions missing from vector index
	Stale       int    `json:"stale"`        // conclusions in vector index with mismatched checksum
	Fresh       int    `json:"fresh"`        // conclusions already indexed and up-to-date
	VectorCount int    `json:"vector_count"` // total entries in vector index
}

// ReindexPreview returns counts of what a reindex would do without mutating.
// It compares active goncho_conclusions against the local vector index by
// memory_id and content checksum. No embedding generation happens during preview.
func (s *Service) ReindexPreview(ctx context.Context) (ReindexPreviewResult, error) {
	if err := ctx.Err(); err != nil {
		return ReindexPreviewResult{}, err
	}
	if s == nil || s.db == nil {
		return ReindexPreviewResult{}, fmt.Errorf("goncho: nil service")
	}

	// Count all non-deleted conclusions for this workspace.
	var total int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM goncho_conclusions
		WHERE workspace_id = ? AND observer_peer_id = ? AND status IN ('processed', 'active')
	`, s.workspaceID, s.observer).Scan(&total)
	if err != nil {
		return ReindexPreviewResult{}, fmt.Errorf("goncho: count conclusions: %w", err)
	}

	// Load conclusion IDs and content checksums for comparison.
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, content FROM goncho_conclusions
		WHERE workspace_id = ? AND observer_peer_id = ? AND status IN ('processed', 'active')
	`, s.workspaceID, s.observer)
	if err != nil {
		return ReindexPreviewResult{}, fmt.Errorf("goncho: list conclusions: %w", err)
	}
	defer rows.Close()

	type conclusionEntry struct {
		id      int64
		content string
	}
	conclusions := make([]conclusionEntry, 0, total)
	for rows.Next() {
		var entry conclusionEntry
		if err := rows.Scan(&entry.id, &entry.content); err != nil {
			return ReindexPreviewResult{}, fmt.Errorf("goncho: scan conclusion: %w", err)
		}
		conclusions = append(conclusions, entry)
	}
	if err := rows.Err(); err != nil {
		return ReindexPreviewResult{}, fmt.Errorf("goncho: iterate conclusions: %w", err)
	}

	// Build map of vector index entries: memory_id -> content_checksum
	vecEntries := map[string]string{} // memory_id -> content_checksum
	if s.vectorStore != nil {
		vecEntries = readVectorIndexEntries(ctx, s.vectorStore)
	}

	notIndexed := 0
	stale := 0
	fresh := 0
	for _, c := range conclusions {
		memID := fmt.Sprintf("%d", c.id)
		checksum := contentChecksum(c.content)
		existing, found := vecEntries[memID]
		if !found {
			notIndexed++
		} else if existing != checksum {
			stale++
		} else {
			fresh++
		}
	}

	return ReindexPreviewResult{
		Status:      "ok",
		Mutates:     false,
		Total:       len(conclusions),
		NotIndexed:  notIndexed,
		Stale:       stale,
		Fresh:       fresh,
		VectorCount: len(vecEntries),
	}, nil
}

// contentChecksum returns the SHA-256 hex checksum of content.
func contentChecksum(content string) string {
	return hashutil.SHA256HexString(strings.TrimSpace(content))
}

// readVectorIndexEntries reads memory_id -> content_checksum from the vector store.
// Falls back to empty map if no vector store or type is unsupported.
func readVectorIndexEntries(ctx context.Context, vs VectorStore) map[string]string {
	entries := map[string]string{}
	// Try type assertion to LocalVectorIndex for direct entry access.
	if lvi, ok := vs.(*LocalVectorIndex); ok {
		lvi.mu.RLock()
		defer lvi.mu.RUnlock()
		for _, entry := range lvi.entries {
			memKey := entry.MemoryID
			if _, exists := entries[memKey]; !exists {
				entries[memKey] = entry.ContentChecksum
			}
		}
	}
	return entries
}

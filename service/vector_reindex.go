package goncho

import (
	"context"
	"fmt"
	"strings"
	"time"

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

type ReindexResult struct {
	ReindexPreviewResult
	Indexed int `json:"indexed"`
	Updated int `json:"updated"`
}

type EmbeddingDiagnosticsReport struct {
	Status      string                      `json:"status"`
	Mutates     bool                        `json:"mutates"`
	Preview     ReindexPreviewResult        `json:"preview"`
	VectorIndex LocalVectorIndexDiagnostics `json:"vector_index,omitempty"`
	Warnings    []string                    `json:"warnings,omitempty"`
}

type reindexConclusionEntry struct {
	id        int64
	peer      string
	content   string
	session   string
	scope     string
	createdAt int64
}

// ReindexPreview returns counts of what a reindex would do without mutating.
// It compares active goncho_conclusions against the local vector index by
// memory_id and content checksum. No embedding generation happens during preview.
func (s *Service) ReindexPreview(ctx context.Context) (ReindexPreviewResult, error) {
	conclusions, err := s.listReindexConclusions(ctx)
	if err != nil {
		return ReindexPreviewResult{}, err
	}
	vecEntries := map[string]string{}
	if s.vectorStore != nil {
		vecEntries = readVectorIndexEntries(ctx, s.vectorStore)
	}
	preview := summarizeReindex(conclusions, vecEntries)
	preview.Status = "ok"
	preview.Mutates = false
	return preview, nil
}

func (s *Service) Reindex(ctx context.Context) (ReindexResult, error) {
	conclusions, err := s.listReindexConclusions(ctx)
	if err != nil {
		return ReindexResult{}, err
	}
	index, ok := s.vectorStore.(*LocalVectorIndex)
	if !ok || index == nil {
		return ReindexResult{}, fmt.Errorf("goncho: local vector index is required for reindex apply")
	}
	before := summarizeReindex(conclusions, readVectorIndexEntries(ctx, index))
	for _, c := range conclusions {
		if beforeEntryFresh(c, readVectorIndexEntries(ctx, index)) {
			continue
		}
		memory := LocalVectorMemory{
			MemoryID:    fmt.Sprintf("%d", c.id),
			WorkspaceID: s.workspaceID,
			Peer:        c.peer,
			SourceType:  "conclusion",
			Content:     c.content,
			SessionID:   c.session,
			ScopeID:     c.scope,
			CreatedAt:   time.Unix(c.createdAt, 0).UTC(),
		}
		if err := index.Upsert(ctx, memory); err != nil {
			return ReindexResult{}, err
		}
	}
	after := summarizeReindex(conclusions, readVectorIndexEntries(ctx, index))
	return ReindexResult{ReindexPreviewResult: ReindexPreviewResult{Status: "ok", Mutates: true, Total: after.Total, NotIndexed: after.NotIndexed, Stale: after.Stale, Fresh: after.Fresh, VectorCount: after.VectorCount}, Indexed: before.NotIndexed, Updated: before.Stale}, nil
}

func (s *Service) EmbeddingDiagnostics(ctx context.Context) (EmbeddingDiagnosticsReport, error) {
	preview, err := s.ReindexPreview(ctx)
	if err != nil {
		return EmbeddingDiagnosticsReport{}, err
	}
	report := EmbeddingDiagnosticsReport{Status: "ok", Mutates: false, Preview: preview}
	if preview.NotIndexed > 0 || preview.Stale > 0 {
		report.Status = "degraded"
		report.Warnings = append(report.Warnings, "local vector index is missing or stale for active conclusions")
	}
	if index, ok := s.vectorStore.(*LocalVectorIndex); ok && index != nil {
		diag, err := index.Diagnostics(ctx)
		if err != nil {
			return EmbeddingDiagnosticsReport{}, err
		}
		report.VectorIndex = diag
		if diag.StaleRows > 0 {
			report.Status = "degraded"
			report.Warnings = append(report.Warnings, "local vector index contains stale rows")
		}
	}
	return report, nil
}

func (s *Service) listReindexConclusions(ctx context.Context) ([]reindexConclusionEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("goncho: nil service")
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, peer_id, content, COALESCE(session_key, ''), scope, created_at FROM goncho_conclusions
		WHERE workspace_id = ? AND observer_peer_id = ? AND status IN ('processed', 'active')
		ORDER BY id ASC
	`, s.workspaceID, s.observer)
	if err != nil {
		return nil, fmt.Errorf("goncho: list conclusions: %w", err)
	}
	defer rows.Close()
	var out []reindexConclusionEntry
	for rows.Next() {
		var entry reindexConclusionEntry
		if err := rows.Scan(&entry.id, &entry.peer, &entry.content, &entry.session, &entry.scope, &entry.createdAt); err != nil {
			return nil, fmt.Errorf("goncho: scan conclusion: %w", err)
		}
		out = append(out, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("goncho: iterate conclusions: %w", err)
	}
	return out, nil
}

func summarizeReindex(conclusions []reindexConclusionEntry, vecEntries map[string]string) ReindexPreviewResult {
	var notIndexed, stale, fresh int
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
	return ReindexPreviewResult{Total: len(conclusions), NotIndexed: notIndexed, Stale: stale, Fresh: fresh, VectorCount: len(vecEntries)}
}

func beforeEntryFresh(c reindexConclusionEntry, vecEntries map[string]string) bool {
	return vecEntries[fmt.Sprintf("%d", c.id)] == contentChecksum(c.content)
}

// contentChecksum returns the SHA-256 hex checksum of content.
func contentChecksum(content string) string {
	return hashutil.SHA256HexString(strings.TrimSpace(content))
}

// readVectorIndexEntries reads memory_id -> content_checksum from the vector store.
// Falls back to empty map if no vector store or type is unsupported.
func readVectorIndexEntries(ctx context.Context, vs VectorStore) map[string]string {
	entries := map[string]string{}
	if ctx.Err() != nil {
		return entries
	}
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

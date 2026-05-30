package goncho

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/textmatch"
)

func (s *Service) ExecuteDreamCompression(ctx context.Context) (int, error) {
	conclusions, err := readAllConclusions(ctx, s.db, s.workspaceID)
	if err != nil {
		return 0, fmt.Errorf("read conclusions: %w", err)
	}

	compressed := 0
	tombstoned := make(map[int64]bool)

	for i := 0; i < len(conclusions); i++ {
		if tombstoned[conclusions[i].ID] {
			continue
		}
		for j := i + 1; j < len(conclusions); j++ {
			if tombstoned[conclusions[j].ID] {
				continue
			}
			similarity := wordSimilarity(conclusions[i].Conclusion, conclusions[j].Conclusion)
			if similarity > 0.6 {
				// Keep the longer one, tombstone the shorter
				if len(conclusions[j].Conclusion) > len(conclusions[i].Conclusion) {
					tombstoned[conclusions[i].ID] = true
					conclusions[i] = conclusions[j]
				} else {
					tombstoned[conclusions[j].ID] = true
				}
				compressed++
			}
		}
	}

	for id := range tombstoned {
		if err := tombstoneConclusion(ctx, s.db, id); err != nil {
			s.log.Warn("dream: failed to tombstone conclusion", "id", id, "err", err)
		}
	}

	return compressed, nil
}

type conclusionEntry struct {
	ID         int64
	Conclusion string
}

func readAllConclusions(ctx context.Context, db *sql.DB, workspaceID string) ([]conclusionEntry, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, content FROM goncho_conclusions
		WHERE workspace_id = ?
		ORDER BY created_at ASC
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []conclusionEntry
	for rows.Next() {
		var e conclusionEntry
		if err := rows.Scan(&e.ID, &e.Conclusion); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func tombstoneConclusion(ctx context.Context, db *sql.DB, id int64) error {
	_, err := db.ExecContext(ctx, `DELETE FROM goncho_conclusions WHERE id = ?`, id)
	return err
}

func wordSimilarity(a, b string) float64 {
	return textmatch.Jaccard(wordSet(a), wordSet(b))
}

func wordSet(s string) map[string]struct{} {
	words := make(map[string]struct{})
	for _, w := range strings.Fields(strings.ToLower(s)) {
		w = strings.Trim(w, ".,;:!?\"'()[]{}")
		if len(w) > 2 {
			words[w] = struct{}{}
		}
	}
	return words
}

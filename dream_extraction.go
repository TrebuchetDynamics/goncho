package goncho

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
)

var conclusionPatternRE = regexp.MustCompile(`(?i)(?:decided|chose|using|switched to|opted for|settled on|prefer|the solution is|we should|must|need to)\s+(.+)`)

func (s *Service) ExecuteDreamFactExtraction(ctx context.Context, sessionKey string) (int, error) {
	turns, err := readSessionTurns(ctx, s.db, s.workspaceID, sessionKey)
	if err != nil {
		return 0, fmt.Errorf("read turns: %w", err)
	}

	extracted := extractConclusionsFromTurns(turns)
	count := 0
	for _, conclusion := range extracted {
		_, err := s.Conclude(ctx, ConcludeParams{
			Peer:       s.observer,
			Conclusion: conclusion,
			SessionKey: sessionKey,
		})
		if err != nil {
			s.log.Warn("dream: failed to write conclusion", "err", err)
			continue
		}
		count++
	}
	return count, nil
}

func readSessionTurns(ctx context.Context, db *sql.DB, workspaceID, sessionKey string) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT content FROM turns
		WHERE session_id = ?
		ORDER BY ts_unix ASC
	`, sessionKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var turns []string
	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			return nil, err
		}
		turns = append(turns, content)
	}
	return turns, rows.Err()
}

func extractConclusionsFromTurns(turns []string) []string {
	seen := make(map[string]bool)
	var conclusions []string

	for _, turn := range turns {
		for _, line := range strings.Split(turn, "\n") {
			line = strings.TrimSpace(line)
			if len(line) < 15 {
				continue
			}
			matches := conclusionPatternRE.FindAllStringSubmatch(line, -1)
			for _, m := range matches {
				if len(m) > 1 {
					conclusion := strings.TrimSpace(m[0])
					if !seen[conclusion] {
						seen[conclusion] = true
						conclusions = append(conclusions, conclusion)
					}
				}
			}
		}
	}
	return conclusions
}

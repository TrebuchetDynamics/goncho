package goncho

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

func Retrieve(ctx context.Context, db *sql.DB, p RetrieveParams) (RetrieveResult, error) {
	if p.Limit <= 0 {
		p.Limit = 10
	}

	ftsHits, err := ftsSearch(ctx, db, p)
	if err != nil {
		return RetrieveResult{}, fmt.Errorf("goncho: fts search: %w", err)
	}

	graphHits, err := graphExpand(ctx, db, ftsHits, p)
	if err != nil {
		return RetrieveResult{}, fmt.Errorf("goncho: graph expand: %w", err)
	}

	candidates := mergeCandidates(ftsHits, graphHits)
	scored := scoreCandidates(candidates, p)
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// MMR diversity
	diverse := mmrSelect(scored, p.Limit, 0.7)

	memories := make([]Memory, 0, len(diverse))
	for _, c := range diverse {
		memories = append(memories, c.memory)
	}

	return RetrieveResult{
		Memories: memories,
		Trace: RetrieveTrace{
			FTSHits:          len(ftsHits),
			GraphHits:        len(graphHits),
			CandidatesScored: len(candidates),
			MMRDiversity:     0.7,
		},
	}, nil
}

type scoredCandidate struct {
	memory Memory
	score  float64
}

func ftsSearch(ctx context.Context, db *sql.DB, p RetrieveParams) ([]Memory, error) {
	args := []any{p.PeerID, p.WorkspaceID}
	where := `m.peer_id = ? AND m.workspace_id = ? AND m.valid_until IS NULL`

	if len(p.Kinds) > 0 {
		placeholders := make([]string, len(p.Kinds))
		for i, k := range p.Kinds {
			placeholders[i] = "?"
			args = append(args, string(k))
		}
		where += fmt.Sprintf(" AND m.kind IN (%s)", strings.Join(placeholders, ","))
	}
	if len(p.Scopes) > 0 {
		placeholders := make([]string, len(p.Scopes))
		for i, s := range p.Scopes {
			placeholders[i] = "?"
			args = append(args, string(s))
		}
		where += fmt.Sprintf(" AND m.scope IN (%s)", strings.Join(placeholders, ","))
	}

	if strings.TrimSpace(p.Query) == "" {
		rows, err := db.QueryContext(ctx, fmt.Sprintf(`
			SELECT m.id, m.kind, m.content, m.peer_id, m.workspace_id, m.scope, m.context_id,
			       m.importance, m.valid_from, m.valid_until, m.supersedes_id,
			       m.created_at, m.updated_at, m.checksum
			FROM memories m
			WHERE %s
			ORDER BY m.updated_at DESC
			LIMIT 200
		`, where), args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanMemories(rows)
	}

	query := buildFTSQuery(p.Query)
	rows, err := db.QueryContext(ctx, fmt.Sprintf(`
		SELECT m.id, m.kind, m.content, m.peer_id, m.workspace_id, m.scope, m.context_id,
		       m.importance, m.valid_from, m.valid_until, m.supersedes_id,
		       m.created_at, m.updated_at, m.checksum
		FROM memory_fts f
		JOIN memories m ON m.id = f.memory_id
		WHERE f.content MATCH ? AND %s
		ORDER BY rank
		LIMIT 200
	`, where), append([]any{query}, args...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMemories(rows)
}

func scanMemories(rows *sql.Rows) ([]Memory, error) {
	var results []Memory
	for rows.Next() {
		var m Memory
		var validUntil sql.NullInt64
		var supersedesID sql.NullString
		var validFrom, createdAt, updatedAt int64
		cols, _ := rows.Columns()
		args := []any{&m.ID, &m.Kind, &m.Content, &m.PeerID, &m.WorkspaceID, &m.Scope,
			&m.ContextID, &m.Importance, &validFrom, &validUntil, &supersedesID,
			&createdAt, &updatedAt, &m.Checksum}
		if len(cols) > 14 {
			var rank float64
			args = append(args, &rank)
		}
		if err := rows.Scan(args...); err != nil {
			return nil, err
		}
		m.ValidFrom = time.Unix(validFrom, 0).UTC()
		if validUntil.Valid {
			m.ValidUntil = time.Unix(validUntil.Int64, 0).UTC()
		}
		if supersedesID.Valid {
			m.SupersedesID = supersedesID.String
		}
		m.CreatedAt = time.Unix(createdAt, 0).UTC()
		m.UpdatedAt = time.Unix(updatedAt, 0).UTC()
		results = append(results, m)
	}
	return results, rows.Err()
}

func buildFTSQuery(query string) string {
	if query == "" {
		return "*"
	}
	terms := strings.Fields(query)
	for i, t := range terms {
		terms[i] = strconv.Quote(t) + "*"
	}
	return strings.Join(terms, " ")
}

func graphExpand(ctx context.Context, db *sql.DB, seeds []Memory, p RetrieveParams) ([]Memory, error) {
	if len(seeds) == 0 {
		return nil, nil
	}

	ids := make([]string, len(seeds))
	for i, s := range seeds {
		ids[i] = s.ID
	}
	placeholders := make([]string, len(ids))
	for i := range ids {
		placeholders[i] = "?"
	}
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	rows, err := db.QueryContext(ctx, fmt.Sprintf(`
		SELECT DISTINCT m.id, m.kind, m.content, m.peer_id, m.workspace_id, m.scope, m.context_id,
		       m.importance, m.valid_from, m.valid_until, m.supersedes_id,
		       m.created_at, m.updated_at, m.checksum
		FROM memory_relations r
		JOIN memories m ON m.id = r.source_id
		WHERE r.target_entity IN (
			SELECT r2.target_entity FROM memory_relations r2 WHERE r2.source_id IN (%s)
		)
		AND m.id NOT IN (%s)
		AND m.valid_until IS NULL
		LIMIT 50
	`, strings.Join(placeholders, ","), strings.Join(placeholders, ",")), append(args, args...)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Memory
	for rows.Next() {
		var m Memory
		var validUntil sql.NullInt64
		var supersedesID sql.NullString
		var validFrom, createdAt, updatedAt int64
		if err := rows.Scan(&m.ID, &m.Kind, &m.Content, &m.PeerID, &m.WorkspaceID, &m.Scope,
			&m.ContextID, &m.Importance, &validFrom, &validUntil, &supersedesID,
			&createdAt, &updatedAt, &m.Checksum); err != nil {
			return nil, err
		}
		m.ValidFrom = time.Unix(validFrom, 0).UTC()
		if validUntil.Valid {
			m.ValidUntil = time.Unix(validUntil.Int64, 0).UTC()
		}
		if supersedesID.Valid {
			m.SupersedesID = supersedesID.String
		}
		m.CreatedAt = time.Unix(createdAt, 0).UTC()
		m.UpdatedAt = time.Unix(updatedAt, 0).UTC()
		results = append(results, m)
	}
	return results, rows.Err()
}

func mergeCandidates(fts, graph []Memory) []Memory {
	seen := make(map[string]bool)
	var all []Memory
	for _, m := range fts {
		if !seen[m.ID] {
			seen[m.ID] = true
			all = append(all, m)
		}
	}
	for _, m := range graph {
		if !seen[m.ID] {
			seen[m.ID] = true
			all = append(all, m)
		}
	}
	return all
}

func scoreCandidates(candidates []Memory, p RetrieveParams) []scoredCandidate {
	now := time.Now()
	scored := make([]scoredCandidate, 0, len(candidates))
	for _, m := range candidates {
		score := 0.0

		// FTS rank boost (FTS hits get higher base score)
		isFTS := false
		for _, c := range candidates {
			if c.ID == m.ID {
				break
			}
		}
		_ = isFTS

		// Importance (0-1)
		score += m.Importance * 0.3

		// Recency decay (exponential, half-life 30 days)
		age := now.Sub(m.UpdatedAt).Hours() / (30 * 24)
		recency := math.Exp(-0.693 * age)
		score += recency * 0.2

		// Goal context boost
		if p.ContextID != "" && m.ContextID == p.ContextID {
			score += 0.3
		}

		// Validity penalty
		if m.IsExpired(now) {
			score *= 0.1
		}

		scored = append(scored, scoredCandidate{memory: m, score: score})
	}
	return scored
}

func mmrSelect(scored []scoredCandidate, limit int, lambda float64) []scoredCandidate {
	if len(scored) <= limit {
		return scored
	}

	selected := []scoredCandidate{scored[0]}
	remaining := scored[1:]

	for len(selected) < limit && len(remaining) > 0 {
		bestIdx := -1
		bestMMR := -math.MaxFloat64

		for i, c := range remaining {
			maxSim := 0.0
			for _, s := range selected {
				sim := contentSimilarity(c.memory.Content, s.memory.Content)
				if sim > maxSim {
					maxSim = sim
				}
			}
			mmr := lambda*c.score - (1-lambda)*maxSim
			if mmr > bestMMR {
				bestMMR = mmr
				bestIdx = i
			}
		}

		if bestIdx >= 0 {
			selected = append(selected, remaining[bestIdx])
			remaining = append(remaining[:bestIdx], remaining[bestIdx+1:]...)
		} else {
			break
		}
	}

	return selected
}

func contentSimilarity(a, b string) float64 {
	wordsA := wordSet(a)
	wordsB := wordSet(b)
	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0
	}
	intersection := 0
	for w := range wordsA {
		if wordsB[w] {
			intersection++
		}
	}
	union := len(wordsA) + len(wordsB) - intersection
	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

func wordSet(s string) map[string]bool {
	words := make(map[string]bool)
	for _, w := range strings.Fields(strings.ToLower(s)) {
		if len(w) > 2 {
			words[w] = true
		}
	}
	return words
}

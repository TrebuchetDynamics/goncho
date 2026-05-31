package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/retrieval"
	"github.com/TrebuchetDynamics/goncho/memory"
	"github.com/TrebuchetDynamics/goncho/service"
)

type LeakageReport struct {
	QueryInMemory  int      `json:"query_in_memory"`
	GoldIDInMemory int      `json:"gold_id_in_memory"`
	Examples       []string `json:"examples,omitempty"`
}

func retrieveForSystem(ctx context.Context, svc *goncho.Service, data dataset, q QuestionRecord, contentIDs map[string][]string, system string, limit int) ([]string, error) {
	switch system {
	case "goncho":
		return retrieveGoncho(ctx, svc, q, contentIDs, limit)
	case "goncho-no-rank":
		return retrieveRecency(data, q, limit), nil
	case "random":
		return retrieveRandom(data, q, limit), nil
	case "bm25":
		return retrieveBM25(data, q, limit), nil
	case "sqlite-fts5":
		return retrieveSQLiteFTS(ctx, data, q, limit)
	default:
		return nil, fmt.Errorf("unknown system %q", system)
	}
}

func retrieveGoncho(ctx context.Context, svc *goncho.Service, q QuestionRecord, contentIDs map[string][]string, limit int) ([]string, error) {
	result, err := svc.Search(ctx, goncho.SearchParams{Peer: q.Peer, SessionKey: q.SessionKey, Query: q.Query, Limit: limit, MaxTokens: 100_000})
	if err != nil {
		return nil, err
	}
	contents := make([]string, 0, len(result.Results))
	for _, hit := range result.Results {
		contents = append(contents, hit.Content)
	}
	return retrieval.StableIDsForContents(q.Peer, contents, contentIDs, 0), nil
}

func contentIDKey(peer, content string) string { return retrieval.ContentKey(peer, content) }

func retrieveRecency(data dataset, q QuestionRecord, limit int) []string {
	out := []string{}
	for i := len(data.Memories) - 1; i >= 0 && len(out) < limit; i-- {
		mem := data.Memories[i]
		if mem.Peer == q.Peer {
			out = append(out, mem.ID)
		}
	}
	return out
}

func retrieveRandom(data dataset, q QuestionRecord, limit int) []string {
	items := peerMemories(data, q.Peer)
	sort.SliceStable(items, func(i, j int) bool {
		return stableHash(q.ID+"/"+items[i].ID) < stableHash(q.ID+"/"+items[j].ID)
	})
	return firstIDs(items, limit)
}

func retrieveBM25(data dataset, q QuestionRecord, limit int) []string {
	items := peerMemories(data, q.Peer)
	return firstIDs(rankMemoriesBM25(q.Query, items), limit)
}

func retrieveSQLiteFTS(ctx context.Context, data dataset, q QuestionRecord, limit int) ([]string, error) {
	dir, err := os.MkdirTemp("", "goncho-bench-fts-*")
	if err != nil {
		return nil, err
	}
	store, err := memory.OpenSqlite(filepath.Join(dir, "fts.db"), 0, nil)
	if err != nil {
		return nil, err
	}
	defer store.Close(ctx)
	db := store.DB()
	if _, err := db.ExecContext(ctx, `CREATE VIRTUAL TABLE bench_fts USING fts5(id UNINDEXED, peer UNINDEXED, content)`); err != nil {
		return nil, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO bench_fts(id, peer, content) VALUES(?, ?, ?)`)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	for _, mem := range data.Memories {
		if mem.Peer == q.Peer {
			if _, err := stmt.ExecContext(ctx, mem.ID, mem.Peer, mem.Content); err != nil {
				_ = stmt.Close()
				_ = tx.Rollback()
				return nil, err
			}
		}
	}
	_ = stmt.Close()
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	query := ftsQuery(q.Query)
	if query == "" {
		return retrieveRecency(data, q, limit), nil
	}
	rows, err := db.QueryContext(ctx, `SELECT id FROM bench_fts WHERE peer = ? AND bench_fts MATCH ? ORDER BY bm25(bench_fts) LIMIT ?`, q.Peer, query, limit)
	if err != nil {
		return retrieveBM25(data, q, limit), nil
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func peerMemories(data dataset, peer string) []MemoryRecord {
	items := []MemoryRecord{}
	for _, mem := range data.Memories {
		if mem.Peer == peer {
			items = append(items, mem)
		}
	}
	return items
}

func firstIDs(items []MemoryRecord, limit int) []string {
	if limit > len(items) {
		limit = len(items)
	}
	out := make([]string, 0, limit)
	for _, item := range items[:limit] {
		out = append(out, item.ID)
	}
	return out
}

func rankMemoriesBM25(query string, items []MemoryRecord) []MemoryRecord {
	records := make([]retrieval.Record, 0, len(items))
	byID := make(map[string]MemoryRecord, len(items))
	for _, item := range items {
		records = append(records, retrieval.Record{ID: item.ID, Peer: item.Peer, Content: item.Content})
		byID[item.ID] = item
	}
	ranked := retrieval.RankBM25(query, records)
	out := make([]MemoryRecord, 0, len(ranked))
	for _, item := range ranked {
		out = append(out, byID[item.ID])
	}
	return out
}

func benchTokenSet(value string) map[string]struct{} { return retrieval.TokenSet(value) }

func benchTokens(value string) []string { return retrieval.Tokens(value) }

func ftsQuery(query string) string { return retrieval.FTSQuery(query) }

func stableHash(value string) uint64 { return retrieval.StableHash(value) }

func checkLeakage(data dataset) LeakageReport {
	report := LeakageReport{Examples: []string{}}
	byPeer := map[string][]MemoryRecord{}
	for _, mem := range data.Memories {
		byPeer[mem.Peer] = append(byPeer[mem.Peer], mem)
	}
	for _, q := range data.Questions {
		query := strings.TrimSpace(strings.ToLower(q.Query))
		gold := set(q.RelevantIDs)
		for _, mem := range byPeer[q.Peer] {
			content := strings.ToLower(mem.Content)
			if query != "" && strings.Contains(content, query) {
				report.QueryInMemory++
				if len(report.Examples) < 10 {
					report.Examples = append(report.Examples, q.ID+":query_in_memory:"+mem.ID)
				}
			}
			for id := range gold {
				if strings.Contains(content, strings.ToLower(id)) {
					report.GoldIDInMemory++
					if len(report.Examples) < 10 {
						report.Examples = append(report.Examples, q.ID+":gold_id_in_memory:"+mem.ID)
					}
				}
			}
		}
	}
	if len(report.Examples) == 0 {
		report.Examples = nil
	}
	return report
}

func writeFailureAudit(path string, report BenchmarkReport) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("goncho-bench: create failures dir: %w", err)
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("goncho-bench: create failures: %w", err)
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	for _, q := range report.Questions {
		if q.Rank == 0 || q.Rank > 10 {
			if err := enc.Encode(q); err != nil {
				return fmt.Errorf("goncho-bench: write failure audit: %w", err)
			}
		}
	}
	return nil
}

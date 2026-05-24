package main

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

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
	retrievedIDs := make([]string, 0, len(result.Results))
	seen := map[string]struct{}{}
	for _, hit := range result.Results {
		for _, id := range contentIDs[contentIDKey(q.Peer, hit.Content)] {
			if _, ok := seen[id]; !ok {
				retrievedIDs = append(retrievedIDs, id)
				seen[id] = struct{}{}
			}
		}
	}
	return retrievedIDs, nil
}

func contentIDKey(peer, content string) string {
	return strings.TrimSpace(peer) + "\x1f" + content
}

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
	queryTokens := benchTokenSet(query)
	if len(queryTokens) == 0 {
		return items
	}
	tfs := make([]map[string]int, len(items))
	lengths := make([]int, len(items))
	df := map[string]int{}
	total := 0
	for i, item := range items {
		tf := benchTermFrequency(item.Content)
		tfs[i] = tf
		for _, count := range tf {
			lengths[i] += count
		}
		total += lengths[i]
		for token := range queryTokens {
			if tf[token] > 0 {
				df[token]++
			}
		}
	}
	avg := 1.0
	if total > 0 {
		avg = float64(total) / float64(len(items))
	}
	type scored struct {
		item  MemoryRecord
		score float64
		index int
	}
	scoredItems := make([]scored, 0, len(items))
	for i, item := range items {
		scoredItems = append(scoredItems, scored{item: item, score: benchBM25Score(queryTokens, tfs[i], df, len(items), lengths[i], avg), index: i})
	}
	sort.SliceStable(scoredItems, func(i, j int) bool {
		if scoredItems[i].score == scoredItems[j].score {
			return scoredItems[i].index < scoredItems[j].index
		}
		return scoredItems[i].score > scoredItems[j].score
	})
	out := make([]MemoryRecord, 0, len(scoredItems))
	for _, item := range scoredItems {
		out = append(out, item.item)
	}
	return out
}

func benchBM25Score(queryTokens map[string]struct{}, tf map[string]int, df map[string]int, docCount, docLength int, avgLength float64) float64 {
	const k1 = 1.2
	const b = 0.75
	if docCount == 0 || docLength == 0 || avgLength <= 0 {
		return 0
	}
	score := 0.0
	for token := range queryTokens {
		freq := tf[token]
		if freq == 0 {
			continue
		}
		idf := math.Log(1 + (float64(docCount)-float64(df[token])+0.5)/(float64(df[token])+0.5))
		denom := float64(freq) + k1*(1-b+b*(float64(docLength)/avgLength))
		score += idf * (float64(freq) * (k1 + 1) / denom)
	}
	return score
}

func benchTermFrequency(value string) map[string]int {
	out := map[string]int{}
	for _, token := range benchTokens(value) {
		out[token]++
	}
	return out
}

func benchTokenSet(value string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, token := range benchTokens(value) {
		out[token] = struct{}{}
	}
	return out
}

var benchTokenPattern = regexp.MustCompile(`[a-z0-9]+`)

func benchTokens(value string) []string {
	out := []string{}
	for _, token := range benchTokenPattern.FindAllString(strings.ToLower(value), -1) {
		token = benchStem(token)
		if len(token) < 3 || benchStopword(token) {
			continue
		}
		out = append(out, token)
	}
	return out
}

func benchStem(token string) string {
	for _, suffix := range []string{"ing", "edly", "ed", "es", "s"} {
		if len(token) > len(suffix)+3 && strings.HasSuffix(token, suffix) {
			return strings.TrimSuffix(token, suffix)
		}
	}
	return token
}

func benchStopword(token string) bool {
	switch token {
	case "the", "and", "for", "who", "what", "when", "where", "which", "should", "not", "did", "does", "with", "that", "this", "from", "are", "was", "were", "has", "have", "had", "you", "your", "about", "can", "could", "would", "there", "their", "they", "them", "then", "than":
		return true
	default:
		return false
	}
}

func ftsQuery(query string) string {
	tokens := benchTokens(query)
	if len(tokens) == 0 {
		return ""
	}
	return strings.Join(tokens, " OR ")
}

func stableHash(value string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(value))
	return h.Sum64()
}

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

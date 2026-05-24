package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/memory"
	"github.com/TrebuchetDynamics/goncho/service"
)

const locomoBenchmarkName = "LOCOMO smoke"

type locomoMemoryRow struct {
	MemoryID       string         `json:"memory_id"`
	ConversationID string         `json:"conversation_id"`
	SessionID      string         `json:"session_id"`
	Speaker        string         `json:"speaker"`
	TurnIndex      int            `json:"turn_index"`
	Timestamp      string         `json:"timestamp"`
	Content        string         `json:"content"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type locomoQuestionRow struct {
	QuestionID     string         `json:"question_id"`
	ConversationID string         `json:"conversation_id"`
	Question       string         `json:"question"`
	GoldMemoryIDs  []string       `json:"gold_memory_ids"`
	Category       string         `json:"category"`
	AnswerHint     string         `json:"answer_hint,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type locomoDataset struct {
	Memories               []locomoMemoryRow
	Questions              []locomoQuestionRow
	memoriesByConversation map[string][]locomoMemoryRow
}

type locomoReport struct {
	BenchmarkName       string               `json:"benchmark_name"`
	Mode                string               `json:"mode"`
	TopK                int                  `json:"top_k"`
	NoLLMJudge          bool                 `json:"no_llm_judge"`
	GeneratedAt         string               `json:"generated_at"`
	RepoCommit          string               `json:"repo_commit,omitempty"`
	GoVersion           string               `json:"go_version"`
	GOOS                string               `json:"goos"`
	GOARCH              string               `json:"goarch"`
	CPUCount            int                  `json:"cpu_count"`
	FixturePaths        locomoFixturePaths   `json:"fixture_paths"`
	Source              map[string]any       `json:"source,omitempty"`
	MemoryCount         int                  `json:"memory_count"`
	MemoryTokenEstimate int                  `json:"memory_token_estimate"`
	DatabaseSizeBytes   int64                `json:"database_size_bytes"`
	QuestionCount       int                  `json:"question_count"`
	LeakageChecks       locomoLeakageChecks  `json:"leakage_checks"`
	Systems             []locomoSystemReport `json:"systems"`
}

type locomoFixturePaths struct {
	Memories  string `json:"memories"`
	Questions string `json:"questions"`
}

type locomoLeakageChecks struct {
	AnswerTextInMemoryContent int      `json:"answer_text_in_memory_content"`
	GoldIDInMemoryContent     int      `json:"gold_id_in_memory_content"`
	QuestionTextInMemory      int      `json:"question_text_in_memory"`
	Examples                  []string `json:"examples,omitempty"`
}

type locomoSystemReport struct {
	System            string                           `json:"system"`
	Questions         int                              `json:"questions"`
	RecallAnyAt5      float64                          `json:"recall_any_at_5"`
	RecallAnyAt10     float64                          `json:"recall_any_at_10"`
	StrictRecallAt5   float64                          `json:"strict_recall_at_5"`
	StrictRecallAt10  float64                          `json:"strict_recall_at_10"`
	NDCGAt5           float64                          `json:"ndcg_at_5"`
	NDCGAt10          float64                          `json:"ndcg_at_10"`
	MRR               float64                          `json:"mrr"`
	SearchLatencyMs   int64                            `json:"search_latency_ms"`
	LatencyMs         locomoLatencyStats               `json:"latency_ms"`
	RSSBytes          uint64                           `json:"rss_bytes"`
	FailureCategories map[string]int                   `json:"failure_categories"`
	CategoryMetrics   map[string]locomoCategoryMetrics `json:"category_metrics"`
	QuestionsDetail   []locomoQuestionResult           `json:"question_results"`
}

type locomoLatencyStats struct {
	Min int64 `json:"min"`
	P50 int64 `json:"p50"`
	P95 int64 `json:"p95"`
	Max int64 `json:"max"`
}

type locomoCategoryMetrics struct {
	Questions        int     `json:"questions"`
	RecallAnyAt5     float64 `json:"recall_any_at_5"`
	RecallAnyAt10    float64 `json:"recall_any_at_10"`
	StrictRecallAt5  float64 `json:"strict_recall_at_5"`
	StrictRecallAt10 float64 `json:"strict_recall_at_10"`
	NDCGAt5          float64 `json:"ndcg_at_5"`
	NDCGAt10         float64 `json:"ndcg_at_10"`
	MRR              float64 `json:"mrr"`
}

type locomoQuestionResult struct {
	QuestionID         string   `json:"question_id"`
	ConversationID     string   `json:"conversation_id"`
	Category           string   `json:"category"`
	Question           string   `json:"question"`
	GoldMemoryIDs      []string `json:"gold_memory_ids"`
	RetrievedIDs       []string `json:"retrieved_ids"`
	Rank               int      `json:"rank"`
	RecallAnyAt5       float64  `json:"recall_any_at_5"`
	RecallAnyAt10      float64  `json:"recall_any_at_10"`
	StrictRecallAt5    float64  `json:"strict_recall_at_5"`
	StrictRecallAt10   float64  `json:"strict_recall_at_10"`
	NDCGAt5            float64  `json:"ndcg_at_5"`
	NDCGAt10           float64  `json:"ndcg_at_10"`
	MRR                float64  `json:"mrr"`
	RetrievalLatencyMs int64    `json:"retrieval_latency_ms"`
}

type locomoFailureRow struct {
	QuestionID      string             `json:"question_id"`
	Category        string             `json:"category"`
	FailureCategory string             `json:"failure_category"`
	FailureBucket   string             `json:"failure_bucket,omitempty"`
	Question        string             `json:"question"`
	GoldMemoryIDs   []string           `json:"gold_memory_ids"`
	TopHits         []locomoFailureHit `json:"top_hits"`
	Notes           string             `json:"notes"`
}

type locomoFailureHit struct {
	Rank      int     `json:"rank"`
	MemoryID  string  `json:"memory_id"`
	Content   string  `json:"content"`
	Score     float64 `json:"score"`
	Speaker   string  `json:"speaker"`
	SessionID string  `json:"session_id"`
	TurnIndex int     `json:"turn_index"`
}

func runLocomoBenchmark(ctx context.Context, cfg config) error {
	if strings.TrimSpace(cfg.LocomoMemoriesPath) == "" || strings.TrimSpace(cfg.LocomoQuestionsPath) == "" {
		return fmt.Errorf("goncho-bench: --locomo-memories and --locomo-questions are required")
	}
	data, err := loadLocomoDataset(cfg.LocomoMemoriesPath, cfg.LocomoQuestionsPath)
	if err != nil {
		return err
	}
	limit := cfg.Limit
	if limit <= 0 {
		limit = 10
	}
	systems := []string{"random", "goncho-no-rank", "recency", "bm25", "sqlite-fts5", "goncho"}
	databaseSizeBytes, err := locomoDatabaseSizeBytes(cfg.LocomoMemoriesPath, cfg.LocomoQuestionsPath)
	if err != nil {
		return err
	}
	reports := make([]locomoSystemReport, 0, len(systems))
	for _, system := range systems {
		systemReport, err := evaluateLocomoSystem(ctx, data, system, limit)
		if err != nil {
			return fmt.Errorf("goncho-bench: locomo %s: %w", system, err)
		}
		reports = append(reports, systemReport)
	}
	benchmarkName := strings.TrimSpace(cfg.LocomoName)
	if benchmarkName == "" {
		benchmarkName = locomoBenchmarkName
	}
	report := locomoReport{
		BenchmarkName:       benchmarkName,
		Mode:                "retrieval",
		TopK:                limit,
		NoLLMJudge:          true,
		GeneratedAt:         time.Now().UTC().Format(time.RFC3339),
		RepoCommit:          gitCommit(),
		GoVersion:           runtime.Version(),
		GOOS:                runtime.GOOS,
		GOARCH:              runtime.GOARCH,
		CPUCount:            runtime.NumCPU(),
		FixturePaths:        locomoFixturePaths{Memories: cfg.LocomoMemoriesPath, Questions: cfg.LocomoQuestionsPath},
		Source:              loadLocomoSourceMetadata(cfg.LocomoMemoriesPath),
		MemoryCount:         len(data.Memories),
		MemoryTokenEstimate: locomoMemoryTokenEstimate(data.Memories),
		DatabaseSizeBytes:   databaseSizeBytes,
		QuestionCount:       len(data.Questions),
		LeakageChecks:       checkLocomoLeakage(data),
		Systems:             reports,
	}
	if err := writeLocomoReport(cfg.OutPath, report); err != nil {
		return err
	}
	if err := writeLocomoFailureAudit(cfg.FailurePath, data, reports); err != nil {
		return err
	}
	if err := writeLocomoMarkdown(cfg.LocomoMarkdownOut, report, cfg.OutPath, cfg.FailurePath); err != nil {
		return err
	}
	return nil
}

func loadLocomoDataset(memoriesPath, questionsPath string) (locomoDataset, error) {
	memories, err := loadLocomoMemories(memoriesPath)
	if err != nil {
		return locomoDataset{}, err
	}
	questions, err := loadLocomoQuestions(questionsPath)
	if err != nil {
		return locomoDataset{}, err
	}
	if err := validateLocomoGoldMemoryIDs(memories, questions); err != nil {
		return locomoDataset{}, err
	}
	return locomoDataset{Memories: memories, Questions: questions, memoriesByConversation: indexLocomoMemoriesByConversation(memories)}, nil
}

func validateLocomoGoldMemoryIDs(memories []locomoMemoryRow, questions []locomoQuestionRow) error {
	memoryConversationIDs := make(map[string]string, len(memories))
	for _, mem := range memories {
		memoryConversationIDs[mem.MemoryID] = mem.ConversationID
	}
	for _, q := range questions {
		seenGoldIDs := map[string]struct{}{}
		for _, id := range q.GoldMemoryIDs {
			if _, exists := seenGoldIDs[id]; exists {
				return fmt.Errorf("goncho-bench: LOCOMO question %q duplicate gold_memory_id %q", q.QuestionID, id)
			}
			seenGoldIDs[id] = struct{}{}
			conversationID, exists := memoryConversationIDs[id]
			if !exists {
				return fmt.Errorf("goncho-bench: LOCOMO question %q unknown gold_memory_id %q", q.QuestionID, id)
			}
			if conversationID != q.ConversationID {
				return fmt.Errorf("goncho-bench: LOCOMO question %q out-of-conversation gold_memory_id %q", q.QuestionID, id)
			}
		}
	}
	return nil
}

func loadLocomoMemories(path string) ([]locomoMemoryRow, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("goncho-bench: open LOCOMO memories: %w", err)
	}
	defer file.Close()
	var out []locomoMemoryRow
	seenIDs := map[string]struct{}{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 32*1024*1024)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var row locomoMemoryRow
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, fmt.Errorf("goncho-bench: decode LOCOMO memory line %d: %w", lineNo, err)
		}
		if strings.TrimSpace(row.MemoryID) == "" || strings.TrimSpace(row.ConversationID) == "" || strings.TrimSpace(row.Content) == "" {
			return nil, fmt.Errorf("goncho-bench: LOCOMO memory line %d missing memory_id/conversation_id/content", lineNo)
		}
		if _, exists := seenIDs[row.MemoryID]; exists {
			return nil, fmt.Errorf("goncho-bench: LOCOMO memory line %d duplicate memory_id %q", lineNo, row.MemoryID)
		}
		seenIDs[row.MemoryID] = struct{}{}
		out = append(out, row)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("goncho-bench: scan LOCOMO memories: %w", err)
	}
	return out, nil
}

func loadLocomoQuestions(path string) ([]locomoQuestionRow, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("goncho-bench: open LOCOMO questions: %w", err)
	}
	defer file.Close()
	var out []locomoQuestionRow
	seenIDs := map[string]struct{}{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 32*1024*1024)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var row locomoQuestionRow
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, fmt.Errorf("goncho-bench: decode LOCOMO question line %d: %w", lineNo, err)
		}
		if strings.TrimSpace(row.QuestionID) == "" || strings.TrimSpace(row.ConversationID) == "" || strings.TrimSpace(row.Question) == "" || len(row.GoldMemoryIDs) == 0 {
			return nil, fmt.Errorf("goncho-bench: LOCOMO question line %d missing question_id/conversation_id/question/gold_memory_ids", lineNo)
		}
		if _, exists := seenIDs[row.QuestionID]; exists {
			return nil, fmt.Errorf("goncho-bench: LOCOMO question line %d duplicate question_id %q", lineNo, row.QuestionID)
		}
		seenIDs[row.QuestionID] = struct{}{}
		out = append(out, row)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("goncho-bench: scan LOCOMO questions: %w", err)
	}
	return out, nil
}

func evaluateLocomoSystem(ctx context.Context, data locomoDataset, system string, limit int) (locomoSystemReport, error) {
	var svc *goncho.Service
	contentIDs := map[string][]string{}
	if system == "goncho" {
		dir, err := os.MkdirTemp("", "goncho-locomo-*")
		if err != nil {
			return locomoSystemReport{}, err
		}
		defer os.RemoveAll(dir)
		store, err := memory.OpenSqlite(filepath.Join(dir, "locomo.db"), 0, nil)
		if err != nil {
			return locomoSystemReport{}, err
		}
		defer store.Close(ctx)
		if err := goncho.RunMigrations(store.DB()); err != nil {
			return locomoSystemReport{}, err
		}
		svc = goncho.NewService(store.DB(), goncho.Config{WorkspaceID: "goncho-locomo-smoke", ObserverPeerID: "goncho-locomo-smoke", RecentMessages: 0}, nil)
		for i, mem := range data.Memories {
			content := locomoIndexableContent(mem)
			result, err := svc.Conclude(ctx, goncho.ConcludeParams{Peer: mem.ConversationID, SessionKey: mem.SessionID, Conclusion: content, Scope: "benchmark"})
			if err != nil {
				return locomoSystemReport{}, err
			}
			if _, err := store.DB().ExecContext(ctx, `UPDATE goncho_conclusions SET created_at = ?, updated_at = ? WHERE id = ?`, i+1, i+1, result.ID); err != nil {
				return locomoSystemReport{}, err
			}
			contentIDs[contentIDKey(mem.ConversationID, content)] = append(contentIDs[contentIDKey(mem.ConversationID, content)], mem.MemoryID)
		}
	}
	searchStart := time.Now()
	results := []locomoQuestionResult{}
	for _, q := range data.Questions {
		questionStart := time.Now()
		ids, err := retrieveLocomo(ctx, svc, data, q, system, contentIDs, limit)
		latencyMs := time.Since(questionStart).Milliseconds()
		if err != nil {
			return locomoSystemReport{}, err
		}
		result := scoreLocomoQuestion(q, ids)
		result.RetrievalLatencyMs = latencyMs
		results = append(results, result)
	}
	report := summarizeLocomoSystem(system, results)
	report.SearchLatencyMs = time.Since(searchStart).Milliseconds()
	report.RSSBytes = currentRSSBytes()
	return report, nil
}

func retrieveLocomo(ctx context.Context, svc *goncho.Service, data locomoDataset, q locomoQuestionRow, system string, contentIDs map[string][]string, limit int) ([]string, error) {
	if limit <= 0 {
		return nil, nil
	}
	items := locomoConversationMemories(data, q.ConversationID)
	switch system {
	case "random":
		sort.SliceStable(items, func(i, j int) bool {
			return stableHash(q.QuestionID+"/"+items[i].MemoryID) < stableHash(q.QuestionID+"/"+items[j].MemoryID)
		})
		return locomoFirstIDs(items, limit), nil
	case "goncho-no-rank", "recency":
		sortLocomoRecency(items)
		return locomoFirstIDs(items, limit), nil
	case "bm25":
		return locomoFirstIDs(rankLocomoBM25(q.Question, items), limit), nil
	case "sqlite-fts5":
		return retrieveLocomoSQLiteFTS(ctx, items, q, limit)
	case "goncho":
		if limit <= 0 {
			return nil, nil
		}
		result, err := svc.Search(ctx, goncho.SearchParams{Peer: q.ConversationID, Query: q.Question, Limit: limit, MaxTokens: 100_000})
		if err != nil {
			return nil, err
		}
		out := []string{}
		seen := map[string]struct{}{}
		for _, hit := range result.Results {
			for _, id := range contentIDs[contentIDKey(q.ConversationID, hit.Content)] {
				if _, ok := seen[id]; !ok {
					seen[id] = struct{}{}
					out = append(out, id)
					if len(out) >= limit {
						return out, nil
					}
				}
			}
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unknown LOCOMO system %q", system)
	}
}

func locomoConversationMemories(data locomoDataset, conversationID string) []locomoMemoryRow {
	if data.memoriesByConversation != nil {
		return append([]locomoMemoryRow(nil), data.memoriesByConversation[conversationID]...)
	}
	out := []locomoMemoryRow{}
	for _, mem := range data.Memories {
		if mem.ConversationID == conversationID {
			out = append(out, mem)
		}
	}
	return out
}

func indexLocomoMemoriesByConversation(memories []locomoMemoryRow) map[string][]locomoMemoryRow {
	byConversation := make(map[string][]locomoMemoryRow, len(memories))
	for _, mem := range memories {
		byConversation[mem.ConversationID] = append(byConversation[mem.ConversationID], mem)
	}
	return byConversation
}

func locomoFirstIDs(items []locomoMemoryRow, limit int) []string {
	if limit > len(items) {
		limit = len(items)
	}
	out := make([]string, 0, limit)
	for _, item := range items[:limit] {
		out = append(out, item.MemoryID)
	}
	return out
}

func sortLocomoRecency(items []locomoMemoryRow) {
	sort.SliceStable(items, func(i, j int) bool {
		ti := parseLocomoTime(items[i].Timestamp)
		tj := parseLocomoTime(items[j].Timestamp)
		if ti != tj {
			return ti > tj
		}
		if items[i].TurnIndex != items[j].TurnIndex {
			return items[i].TurnIndex > items[j].TurnIndex
		}
		return items[i].MemoryID > items[j].MemoryID
	})
}

func parseLocomoTime(value string) int64 {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return 0
	}
	return t.Unix()
}

func rankLocomoBM25(query string, items []locomoMemoryRow) []locomoMemoryRow {
	records := make([]MemoryRecord, 0, len(items))
	byID := map[string]locomoMemoryRow{}
	for _, item := range items {
		records = append(records, MemoryRecord{ID: item.MemoryID, Peer: item.ConversationID, Content: locomoIndexableContent(item)})
		byID[item.MemoryID] = item
	}
	ranked := rankMemoriesBM25(query, records)
	out := make([]locomoMemoryRow, 0, len(ranked))
	for _, item := range ranked {
		out = append(out, byID[item.ID])
	}
	return out
}

func retrieveLocomoSQLiteFTS(ctx context.Context, items []locomoMemoryRow, q locomoQuestionRow, limit int) ([]string, error) {
	query := ftsQuery(q.Question)
	if query == "" {
		sortLocomoRecency(items)
		return locomoFirstIDs(items, limit), nil
	}
	dir, err := os.MkdirTemp("", "goncho-locomo-fts-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	store, err := memory.OpenSqlite(filepath.Join(dir, "fts.db"), 0, nil)
	if err != nil {
		return nil, err
	}
	defer store.Close(ctx)
	db := store.DB()
	if _, err := db.ExecContext(ctx, `CREATE VIRTUAL TABLE locomo_fts USING fts5(id UNINDEXED, content)`); err != nil {
		return nil, err
	}
	if err := insertLocomoFTSRows(ctx, db, items); err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `SELECT id FROM locomo_fts WHERE locomo_fts MATCH ? ORDER BY bm25(locomo_fts) LIMIT ?`, query, limit)
	if err != nil {
		return locomoFirstIDs(rankLocomoBM25(q.Question, items), limit), nil
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
	return out, rows.Err()
}

func insertLocomoFTSRows(ctx context.Context, db *sql.DB, items []locomoMemoryRow) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO locomo_fts(id, content) VALUES(?, ?)`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	for _, item := range items {
		if _, err := stmt.ExecContext(ctx, item.MemoryID, locomoIndexableContent(item)); err != nil {
			_ = stmt.Close()
			_ = tx.Rollback()
			return err
		}
	}
	_ = stmt.Close()
	return tx.Commit()
}

func locomoIndexableContent(mem locomoMemoryRow) string {
	parts := []string{
		"speaker: " + strings.TrimSpace(mem.Speaker),
		"timestamp: " + strings.TrimSpace(mem.Timestamp),
		"session: " + strings.TrimSpace(mem.SessionID),
		fmt.Sprintf("turn: %d", mem.TurnIndex),
		"content: " + strings.TrimSpace(mem.Content),
	}
	return strings.Join(parts, "\n")
}

func locomoMemoryTokenEstimate(memories []locomoMemoryRow) int {
	total := 0
	for _, mem := range memories {
		total += len(benchTokenPattern.FindAllString(strings.ToLower(mem.Content), -1))
	}
	return total
}

func locomoDatabaseSizeBytes(paths ...string) (int64, error) {
	var total int64
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return 0, fmt.Errorf("goncho-bench: stat LOCOMO fixture %q: %w", path, err)
		}
		total += info.Size()
	}
	return total, nil
}

func scoreLocomoQuestion(q locomoQuestionRow, retrieved []string) locomoQuestionResult {
	rank := firstRelevantRank(retrieved, q.GoldMemoryIDs)
	mrr := 0.0
	if rank > 0 {
		mrr = roundMetric(1 / float64(rank))
	}
	return locomoQuestionResult{
		QuestionID:       q.QuestionID,
		ConversationID:   q.ConversationID,
		Category:         q.Category,
		Question:         q.Question,
		GoldMemoryIDs:    append([]string(nil), q.GoldMemoryIDs...),
		RetrievedIDs:     append([]string(nil), retrieved...),
		Rank:             rank,
		RecallAnyAt5:     locomoRecallAny(retrieved, q.GoldMemoryIDs, 5),
		RecallAnyAt10:    locomoRecallAny(retrieved, q.GoldMemoryIDs, 10),
		StrictRecallAt5:  locomoStrictRecall(retrieved, q.GoldMemoryIDs, 5),
		StrictRecallAt10: locomoStrictRecall(retrieved, q.GoldMemoryIDs, 10),
		NDCGAt5:          locomoNDCG(retrieved, q.GoldMemoryIDs, 5),
		NDCGAt10:         locomoNDCG(retrieved, q.GoldMemoryIDs, 10),
		MRR:              mrr,
	}
}

func locomoRecallAny(retrieved, gold []string, k int) float64 {
	seen := map[string]struct{}{}
	for _, id := range retrieved[:min(k, len(retrieved))] {
		seen[id] = struct{}{}
	}
	for _, id := range gold {
		if _, ok := seen[id]; ok {
			return 1
		}
	}
	return 0
}

func locomoStrictRecall(retrieved, gold []string, k int) float64 {
	seen := map[string]struct{}{}
	for _, id := range retrieved[:min(k, len(retrieved))] {
		seen[id] = struct{}{}
	}
	for _, id := range gold {
		if _, ok := seen[id]; !ok {
			return 0
		}
	}
	return 1
}

func locomoNDCG(retrieved, gold []string, k int) float64 {
	if k <= 0 || len(gold) == 0 {
		return 0
	}
	goldSet := map[string]struct{}{}
	for _, id := range gold {
		goldSet[id] = struct{}{}
	}
	seenRelevant := map[string]struct{}{}
	dcg := 0.0
	for i, id := range retrieved[:min(k, len(retrieved))] {
		if _, ok := goldSet[id]; !ok {
			continue
		}
		if _, ok := seenRelevant[id]; ok {
			continue
		}
		seenRelevant[id] = struct{}{}
		dcg += 1 / math.Log2(float64(i+2))
	}
	idealCount := min(k, len(goldSet))
	idcg := 0.0
	for i := 0; i < idealCount; i++ {
		idcg += 1 / math.Log2(float64(i+2))
	}
	if idcg == 0 {
		return 0
	}
	return roundMetric(dcg / idcg)
}

func summarizeLocomoSystem(system string, results []locomoQuestionResult) locomoSystemReport {
	out := locomoSystemReport{System: system, Questions: len(results), FailureCategories: locomoFailureCategories(results), CategoryMetrics: map[string]locomoCategoryMetrics{}, QuestionsDetail: results}
	if len(results) == 0 {
		return out
	}
	var any5, any10, strict5, strict10, ndcg5, ndcg10, mrr float64
	byCategory := map[string][]locomoQuestionResult{}
	for _, q := range results {
		any5 += q.RecallAnyAt5
		any10 += q.RecallAnyAt10
		strict5 += q.StrictRecallAt5
		strict10 += q.StrictRecallAt10
		ndcg5 += q.NDCGAt5
		ndcg10 += q.NDCGAt10
		mrr += q.MRR
		byCategory[q.Category] = append(byCategory[q.Category], q)
	}
	out.RecallAnyAt5 = roundMetric(any5 / float64(len(results)))
	out.RecallAnyAt10 = roundMetric(any10 / float64(len(results)))
	out.StrictRecallAt5 = roundMetric(strict5 / float64(len(results)))
	out.StrictRecallAt10 = roundMetric(strict10 / float64(len(results)))
	out.NDCGAt5 = roundMetric(ndcg5 / float64(len(results)))
	out.NDCGAt10 = roundMetric(ndcg10 / float64(len(results)))
	out.MRR = roundMetric(mrr / float64(len(results)))
	out.LatencyMs = summarizeLocomoLatency(results)
	for category, items := range byCategory {
		out.CategoryMetrics[category] = summarizeLocomoCategory(items)
	}
	return out
}

func summarizeLocomoCategory(results []locomoQuestionResult) locomoCategoryMetrics {
	var any5, any10, strict5, strict10, ndcg5, ndcg10, mrr float64
	for _, q := range results {
		any5 += q.RecallAnyAt5
		any10 += q.RecallAnyAt10
		strict5 += q.StrictRecallAt5
		strict10 += q.StrictRecallAt10
		ndcg5 += q.NDCGAt5
		ndcg10 += q.NDCGAt10
		mrr += q.MRR
	}
	n := float64(len(results))
	return locomoCategoryMetrics{Questions: len(results), RecallAnyAt5: roundMetric(any5 / n), RecallAnyAt10: roundMetric(any10 / n), StrictRecallAt5: roundMetric(strict5 / n), StrictRecallAt10: roundMetric(strict10 / n), NDCGAt5: roundMetric(ndcg5 / n), NDCGAt10: roundMetric(ndcg10 / n), MRR: roundMetric(mrr / n)}
}

func summarizeLocomoLatency(results []locomoQuestionResult) locomoLatencyStats {
	if len(results) == 0 {
		return locomoLatencyStats{}
	}
	values := make([]int64, 0, len(results))
	for _, q := range results {
		values = append(values, q.RetrievalLatencyMs)
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	return locomoLatencyStats{
		Min: values[0],
		P50: locomoNearestRankLatency(values, 50),
		P95: locomoNearestRankLatency(values, 95),
		Max: values[len(values)-1],
	}
}

func locomoNearestRankLatency(sortedValues []int64, percentile int) int64 {
	if len(sortedValues) == 0 {
		return 0
	}
	rank := (len(sortedValues)*percentile + 99) / 100
	if rank < 1 {
		rank = 1
	}
	if rank > len(sortedValues) {
		rank = len(sortedValues)
	}
	return sortedValues[rank-1]
}

func writeLocomoReport(path string, report locomoReport) error {
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if strings.TrimSpace(path) == "" {
		_, err = os.Stdout.Write(raw)
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func writeLocomoFailureAudit(path string, data locomoDataset, reports []locomoSystemReport) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	var gonchoReport *locomoSystemReport
	for i := range reports {
		if reports[i].System == "goncho" {
			gonchoReport = &reports[i]
			break
		}
	}
	if gonchoReport == nil {
		return nil
	}
	memByID := map[string]locomoMemoryRow{}
	memoryConversationIDs := map[string]string{}
	for _, mem := range data.Memories {
		memByID[mem.MemoryID] = mem
		memoryConversationIDs[mem.MemoryID] = mem.ConversationID
	}
	questionsByID := map[string]locomoQuestionRow{}
	for _, q := range data.Questions {
		questionsByID[q.QuestionID] = q
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(file)
	for _, q := range gonchoReport.QuestionsDetail {
		if locomoFailureAuditShouldSkip(q) {
			continue
		}
		fixtureQuestion, ok := questionsByID[q.QuestionID]
		if !ok {
			_ = file.Close()
			return fmt.Errorf("goncho-bench: LOCOMO failure audit unknown question_id %q", q.QuestionID)
		}
		if q.ConversationID != fixtureQuestion.ConversationID {
			_ = file.Close()
			return fmt.Errorf("goncho-bench: LOCOMO failure audit question_id %q conversation_id %q does not match fixture conversation_id %q", q.QuestionID, q.ConversationID, fixtureQuestion.ConversationID)
		}
		for _, id := range q.GoldMemoryIDs {
			mem, ok := memByID[id]
			if !ok {
				_ = file.Close()
				return fmt.Errorf("goncho-bench: LOCOMO failure audit question %q unknown gold_memory_id %q", q.QuestionID, id)
			}
			if mem.ConversationID != q.ConversationID {
				_ = file.Close()
				return fmt.Errorf("goncho-bench: LOCOMO failure audit question %q out-of-conversation gold_memory_id %q", q.QuestionID, id)
			}
		}
		row := locomoFailureRow{QuestionID: q.QuestionID, Category: q.Category, FailureCategory: q.Category, FailureBucket: classifyLocomoFailureBucket(q, memoryConversationIDs), Question: q.Question, GoldMemoryIDs: q.GoldMemoryIDs, Notes: locomoFailureNotes(q)}
		for i, id := range q.RetrievedIDs[:min(10, len(q.RetrievedIDs))] {
			mem, ok := memByID[id]
			if !ok {
				_ = file.Close()
				return fmt.Errorf("goncho-bench: LOCOMO failure audit question %q unknown retrieved memory_id %q", q.QuestionID, id)
			}
			if mem.ConversationID != q.ConversationID {
				_ = file.Close()
				return fmt.Errorf("goncho-bench: LOCOMO failure audit question %q out-of-conversation retrieved memory_id %q", q.QuestionID, id)
			}
			row.TopHits = append(row.TopHits, locomoFailureHit{Rank: i + 1, MemoryID: id, Content: mem.Content, Score: 0, Speaker: mem.Speaker, SessionID: mem.SessionID, TurnIndex: mem.TurnIndex})
		}
		if err := enc.Encode(row); err != nil {
			_ = file.Close()
			return err
		}
	}
	return file.Close()
}

func locomoFailureAuditShouldSkip(q locomoQuestionResult) bool {
	return q.Rank == 1 && (q.StrictRecallAt10 == 1 || len(q.GoldMemoryIDs) <= 1)
}

func locomoFailureNotes(q locomoQuestionResult) string {
	if q.Rank == 0 {
		return fmt.Sprintf("no gold memory ID appeared in top %d", len(q.RetrievedIDs))
	}
	return fmt.Sprintf("first gold memory ID appeared at rank %d", q.Rank)
}

func writeLocomoMarkdown(path string, report locomoReport, jsonPath, failurePath string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s Retrieval Report\n\n", report.BenchmarkName)
	if strings.Contains(strings.ToLower(report.BenchmarkName), "smoke") {
		b.WriteString("LOCOMO smoke validates the benchmark harness. It is not a publishable full benchmark result.\n\n")
	} else {
		b.WriteString("This is the full pinned LOCOMO retrieval benchmark report generated by Goncho's deterministic harness.\n\n")
	}
	b.WriteString("This evaluates retrieval, not answer generation. It uses deterministic ID-based scoring and no LLM judge. `answer_hint` fields are never indexed or scored.\n\n")
	fmt.Fprintf(&b, "- JSON evidence: `%s`\n", jsonPath)
	fmt.Fprintf(&b, "- Failure JSONL: `%s`\n", failurePath)
	label := "fixture"
	if !strings.Contains(strings.ToLower(report.BenchmarkName), "smoke") {
		label = "converted dataset"
	}
	fmt.Fprintf(&b, "- Memories %s: `%s`\n", label, report.FixturePaths.Memories)
	fmt.Fprintf(&b, "- Questions %s: `%s`\n", label, report.FixturePaths.Questions)
	fmt.Fprintf(&b, "- Questions: `%d`\n", report.QuestionCount)
	fmt.Fprintf(&b, "- Memories: `%d`\n", report.MemoryCount)
	fmt.Fprintf(&b, "- Memory token estimate: `%d`\n", report.MemoryTokenEstimate)
	fmt.Fprintf(&b, "- Database size bytes: `%d`\n", report.DatabaseSizeBytes)
	fmt.Fprintf(&b, "- Mode: `%s`\n", report.Mode)
	fmt.Fprintf(&b, "- Top-K: `%d`\n", report.TopK)
	fmt.Fprintf(&b, "- no_llm_judge: `%t`\n", report.NoLLMJudge)
	fmt.Fprintf(&b, "- Reproduce: `go run ./cmd/goncho-bench --locomo-memories %s --locomo-questions %s --out %s --failures %s --locomo-md-out %s --limit %d`\n", report.FixturePaths.Memories, report.FixturePaths.Questions, jsonPath, failurePath, path, report.TopK)
	if len(report.Source) > 0 {
		fmt.Fprintf(&b, "- Source: `%v` at `%v`\n", report.Source["source_url"], report.Source["source_revision"])
		fmt.Fprintf(&b, "- Source SHA256: `%v`\n", report.Source["source_sha256"])
		if value, ok := report.Source["converted_memories_sha256"]; ok {
			fmt.Fprintf(&b, "- Converted memories SHA256: `%v`\n", value)
		}
		if value, ok := report.Source["converted_questions_sha256"]; ok {
			fmt.Fprintf(&b, "- Converted questions SHA256: `%v`\n", value)
		}
		fmt.Fprintf(&b, "- License note: `%v`\n", report.Source["license"])
	}
	b.WriteString("\n")
	b.WriteString("## Systems\n\n| System | recall_any@5 | recall_any@10 | strict_recall@5 | strict_recall@10 | NDCG@5 | NDCG@10 | MRR | Search latency ms | Latency min ms | Latency p50 ms | Latency p95 ms | Latency max ms | RSS bytes |\n| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	for _, system := range report.Systems {
		fmt.Fprintf(&b, "| %s | %.2f%% | %.2f%% | %.2f%% | %.2f%% | %.2f%% | %.2f%% | %.2f%% | %d | %d | %d | %d | %d | %d |\n", system.System, system.RecallAnyAt5*100, system.RecallAnyAt10*100, system.StrictRecallAt5*100, system.StrictRecallAt10*100, system.NDCGAt5*100, system.NDCGAt10*100, system.MRR*100, system.SearchLatencyMs, system.LatencyMs.Min, system.LatencyMs.P50, system.LatencyMs.P95, system.LatencyMs.Max, system.RSSBytes)
	}
	b.WriteString("\n## Failure categories\n\n| System | Category | Questions |\n| --- | --- | ---: |\n")
	for _, system := range report.Systems {
		for _, category := range sortedLocomoFailureCategories(system.FailureCategories) {
			fmt.Fprintf(&b, "| %s | `%s` | %d |\n", system.System, category, system.FailureCategories[category])
		}
	}
	b.WriteString("\n## Category metrics\n\n")
	for _, system := range report.Systems {
		fmt.Fprintf(&b, "### %s\n\n| Category | Questions | recall_any@5 | recall_any@10 | strict_recall@5 | strict_recall@10 | NDCG@5 | NDCG@10 | MRR |\n| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |\n", system.System)
		for _, category := range sortedLocomoCategories(system.CategoryMetrics) {
			m := system.CategoryMetrics[category]
			fmt.Fprintf(&b, "| `%s` | %d | %.2f%% | %.2f%% | %.2f%% | %.2f%% | %.2f%% | %.2f%% | %.2f%% |\n", category, m.Questions, m.RecallAnyAt5*100, m.RecallAnyAt10*100, m.StrictRecallAt5*100, m.StrictRecallAt10*100, m.NDCGAt5*100, m.NDCGAt10*100, m.MRR*100)
		}
		b.WriteString("\n")
	}
	b.WriteString("## Leakage checks\n\n")
	fmt.Fprintf(&b, "- Answer text present in memory content: `%d`\n", report.LeakageChecks.AnswerTextInMemoryContent)
	fmt.Fprintf(&b, "- Gold IDs present in memory content: `%d`\n", report.LeakageChecks.GoldIDInMemoryContent)
	fmt.Fprintf(&b, "- Question text present in memory content: `%d`\n\n", report.LeakageChecks.QuestionTextInMemory)
	b.WriteString("`answer_hint` is not indexed or scored. Answer-text presence is reported because LOCOMO answers may be literal spans from the gold memories.\n\n")
	b.WriteString("## Notes\n\n- Retrieval-first only.\n- No answer generation.\n- No LLM judge.\n- Baselines included: random, Goncho no-rank, recency, BM25, SQLite FTS5, Goncho current.\n")
	if strings.Contains(strings.ToLower(report.BenchmarkName), "smoke") {
		b.WriteString("- The smoke fixture intentionally includes latest-state, historical, speaker-attribution, contradiction/supersession, multi-session, lexical miss, gold ambiguity, and true retrieval failure categories.\n")
	} else {
		b.WriteString("- This full run uses the pinned official LOCOMO dataset conversion described in `docs/benchmarks/LOCOMO-DATASET.md`.\n")
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func sortedLocomoCategories(metrics map[string]locomoCategoryMetrics) []string {
	keys := make([]string, 0, len(metrics))
	for key := range metrics {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedLocomoFailureCategories(metrics map[string]int) []string {
	keys := make([]string, 0, len(metrics))
	for key := range metrics {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func checkLocomoLeakage(data locomoDataset) locomoLeakageChecks {
	checks := locomoLeakageChecks{}
	byConversation := data.memoriesByConversation
	if byConversation == nil {
		byConversation = indexLocomoMemoriesByConversation(data.Memories)
	}
	for _, q := range data.Questions {
		answer := strings.TrimSpace(strings.ToLower(q.AnswerHint))
		question := strings.TrimSpace(strings.ToLower(q.Question))
		for _, mem := range byConversation[q.ConversationID] {
			content := strings.ToLower(mem.Content)
			if answer != "" && strings.Contains(content, answer) {
				checks.AnswerTextInMemoryContent++
				checks.addExample("answer_text", q.QuestionID, mem.MemoryID)
			}
			if question != "" && strings.Contains(content, question) {
				checks.QuestionTextInMemory++
				checks.addExample("question_text", q.QuestionID, mem.MemoryID)
			}
			for _, gold := range q.GoldMemoryIDs {
				if strings.Contains(content, strings.ToLower(gold)) {
					checks.GoldIDInMemoryContent++
					checks.addExample("gold_id", q.QuestionID, mem.MemoryID)
				}
			}
		}
	}
	return checks
}

func (c *locomoLeakageChecks) addExample(kind, questionID, memoryID string) {
	if len(c.Examples) < 10 {
		c.Examples = append(c.Examples, kind+":"+questionID+":"+memoryID)
	}
}

func loadLocomoSourceMetadata(memoriesPath string) map[string]any {
	path := filepath.Join(filepath.Dir(memoriesPath), "metadata.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var meta map[string]any
	if err := json.Unmarshal(raw, &meta); err != nil {
		return nil
	}
	return meta
}

func gitCommit() string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	raw, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}

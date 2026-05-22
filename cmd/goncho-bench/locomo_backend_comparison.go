package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho"
	"github.com/TrebuchetDynamics/goncho/memory"
)

type MemoryBackend interface {
	Name() string
	Reset(context.Context) error
	Insert(context.Context, string, string, map[string]any) error
	Search(context.Context, string, int) ([]BackendResult, error)
	Close(context.Context) error
}

type BackendResult struct {
	MemoryID string  `json:"memory_id"`
	Score    float64 `json:"score"`
}

type scopedMemoryBackend interface {
	SearchScoped(context.Context, string, string, int) ([]BackendResult, error)
}

type locomoBackendComparisonReport struct {
	BenchmarkName string                         `json:"benchmark_name"`
	Mode          string                         `json:"mode"`
	NoLLMJudge    bool                           `json:"no_llm_judge"`
	GeneratedAt   string                         `json:"generated_at"`
	RepoCommit    string                         `json:"repo_commit,omitempty"`
	GoVersion     string                         `json:"go_version"`
	GOOS          string                         `json:"goos"`
	GOARCH        string                         `json:"goarch"`
	CPUCount      int                            `json:"cpu_count"`
	FixturePaths  locomoFixturePaths             `json:"fixture_paths"`
	Source        map[string]any                 `json:"source,omitempty"`
	Rules         []string                       `json:"rules"`
	MemoryCount   int                            `json:"memory_count"`
	QuestionCount int                            `json:"question_count"`
	Backends      []locomoBackendComparisonEntry `json:"backends"`
}

type locomoBackendComparisonEntry struct {
	Backend             string                 `json:"backend"`
	Comparable          bool                   `json:"comparable"`
	NotComparableReason string                 `json:"not_comparable_reason,omitempty"`
	Questions           int                    `json:"questions,omitempty"`
	RecallAnyAt5        float64                `json:"recall_any_at_5,omitempty"`
	RecallAnyAt10       float64                `json:"recall_any_at_10,omitempty"`
	StrictRecallAt5     float64                `json:"strict_recall_at_5,omitempty"`
	StrictRecallAt10    float64                `json:"strict_recall_at_10,omitempty"`
	MRR                 float64                `json:"mrr,omitempty"`
	InsertLatencyMs     int64                  `json:"insert_latency_ms,omitempty"`
	SearchLatencyMs     int64                  `json:"search_latency_ms,omitempty"`
	RSSBytes            uint64                 `json:"rss_bytes,omitempty"`
	FailureCategories   map[string]int         `json:"failure_categories,omitempty"`
	QuestionsDetail     []locomoQuestionResult `json:"question_results,omitempty"`
	SetupNotes          []string               `json:"setup_notes,omitempty"`
}

type externalLocomoAdapterRow struct {
	Backend             string                 `json:"backend"`
	QuestionID          string                 `json:"question_id,omitempty"`
	Comparable          bool                   `json:"comparable"`
	NotComparableReason string                 `json:"not_comparable_reason,omitempty"`
	Reason              string                 `json:"reason,omitempty"`
	Results             []externalLocomoResult `json:"results,omitempty"`
	SetupNotes          []string               `json:"setup_notes,omitempty"`
}

type externalLocomoResult struct {
	MemoryID     string         `json:"memory_id"`
	Score        float64        `json:"score"`
	BackendRawID string         `json:"backend_raw_id,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

func runLocomoBackendComparison(ctx context.Context, cfg config) error {
	if strings.TrimSpace(cfg.LocomoMemoriesPath) == "" || strings.TrimSpace(cfg.LocomoQuestionsPath) == "" {
		return fmt.Errorf("goncho-bench: --locomo-memories and --locomo-questions are required for backend comparison")
	}
	data, err := loadLocomoDataset(cfg.LocomoMemoriesPath, cfg.LocomoQuestionsPath)
	if err != nil {
		return err
	}
	backends := []string{"goncho", "bm25", "sqlite-fts5", "agentmemory", "mem0"}
	entries := make([]locomoBackendComparisonEntry, 0, len(backends))
	for _, name := range backends {
		entry, err := evaluateLocomoBackend(ctx, data, name, 10, cfg)
		if err != nil {
			return fmt.Errorf("goncho-bench: backend %s: %w", name, err)
		}
		entries = append(entries, entry)
	}
	report := locomoBackendComparisonReport{
		BenchmarkName: "LOCOMO backend comparison",
		Mode:          "retrieval_backend_adapter",
		NoLLMJudge:    true,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		RepoCommit:    gitCommit(),
		GoVersion:     runtime.Version(), GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, CPUCount: runtime.NumCPU(),
		FixturePaths: locomoFixturePaths{Memories: cfg.LocomoMemoriesPath, Questions: cfg.LocomoQuestionsPath},
		Source:       loadLocomoSourceMetadata(cfg.LocomoMemoriesPath),
		Rules: []string{
			"retrieval only",
			"no LLM judge",
			"no answer generation",
			"same converted memories/questions",
			"same gold memory IDs",
			"same top-K scoring",
			"if stable memory IDs are unavailable, mark backend not comparable",
		},
		MemoryCount: len(data.Memories), QuestionCount: len(data.Questions), Backends: entries,
	}
	if err := writeLocomoBackendComparisonJSON(cfg.LocomoBackendComparisonJSON, report); err != nil {
		return err
	}
	if err := writeLocomoBackendComparisonFailures(cfg.LocomoBackendComparisonFailures, data, report); err != nil {
		return err
	}
	return writeLocomoBackendComparisonMarkdown(cfg.LocomoBackendComparisonMD, report, cfg.LocomoBackendComparisonJSON, cfg.LocomoBackendComparisonFailures)
}

func evaluateLocomoBackend(ctx context.Context, data locomoDataset, name string, topK int, cfg config) (locomoBackendComparisonEntry, error) {
	if path := externalLocomoResultsPath(name, cfg); path != "" {
		return evaluateExternalLocomoResults(data, name, path)
	}
	backend, unsupported, err := newLocomoBackend(name)
	if err != nil {
		return locomoBackendComparisonEntry{}, err
	}
	if unsupported != "" {
		return locomoBackendComparisonEntry{Backend: name, Comparable: false, NotComparableReason: unsupported, SetupNotes: setupNotesForBackend(name)}, nil
	}
	defer backend.Close(ctx)
	if err := backend.Reset(ctx); err != nil {
		return locomoBackendComparisonEntry{}, err
	}
	insertStart := time.Now()
	for _, mem := range data.Memories {
		if err := backend.Insert(ctx, mem.MemoryID, locomoIndexableContent(mem), locomoMemoryMetadata(mem)); err != nil {
			return locomoBackendComparisonEntry{}, err
		}
	}
	insertLatency := time.Since(insertStart).Milliseconds()
	searchStart := time.Now()
	results := make([]locomoQuestionResult, 0, len(data.Questions))
	for _, q := range data.Questions {
		hits, err := searchLocomoBackend(ctx, backend, q, topK)
		if err != nil {
			return locomoBackendComparisonEntry{}, err
		}
		ids := make([]string, 0, len(hits))
		for _, hit := range hits {
			ids = append(ids, hit.MemoryID)
		}
		results = append(results, scoreLocomoQuestion(q, ids))
	}
	searchLatency := time.Since(searchStart).Milliseconds()
	summary := summarizeLocomoSystem(name, results)
	return locomoBackendComparisonEntry{
		Backend: name, Comparable: true, Questions: len(results),
		RecallAnyAt5: summary.RecallAnyAt5, RecallAnyAt10: summary.RecallAnyAt10,
		StrictRecallAt5: summary.StrictRecallAt5, StrictRecallAt10: summary.StrictRecallAt10, MRR: summary.MRR,
		InsertLatencyMs: insertLatency, SearchLatencyMs: searchLatency, RSSBytes: currentRSSBytes(),
		FailureCategories: locomoFailureCategories(results), QuestionsDetail: results, SetupNotes: setupNotesForBackend(name),
	}, nil
}

func externalLocomoResultsPath(name string, cfg config) string {
	switch name {
	case "agentmemory":
		return strings.TrimSpace(cfg.LocomoAgentMemoryResults)
	case "mem0":
		return strings.TrimSpace(cfg.LocomoMem0Results)
	default:
		return ""
	}
}

func evaluateExternalLocomoResults(data locomoDataset, name, path string) (locomoBackendComparisonEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return locomoBackendComparisonEntry{}, fmt.Errorf("read external %s results: %w", name, err)
	}
	defer file.Close()
	rowsByQuestion := map[string]externalLocomoAdapterRow{}
	statusReason := ""
	setupNotes := setupNotesForBackend(name)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024*1024), 32*1024*1024)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var row externalLocomoAdapterRow
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return locomoBackendComparisonEntry{}, fmt.Errorf("decode external %s results line %d: %w", name, lineNo, err)
		}
		if row.Backend != "" && row.Backend != name {
			return locomoBackendComparisonEntry{}, fmt.Errorf("external results backend mismatch line %d: got %q want %q", lineNo, row.Backend, name)
		}
		if len(row.SetupNotes) > 0 {
			setupNotes = row.SetupNotes
		}
		if !row.Comparable {
			if row.NotComparableReason != "" {
				statusReason = row.NotComparableReason
			} else if row.Reason != "" {
				statusReason = row.Reason
			}
			continue
		}
		if row.QuestionID == "" {
			return locomoBackendComparisonEntry{}, fmt.Errorf("external %s comparable row line %d missing question_id", name, lineNo)
		}
		if _, exists := rowsByQuestion[row.QuestionID]; exists {
			return locomoBackendComparisonEntry{}, fmt.Errorf("external %s duplicate question_id %q", name, row.QuestionID)
		}
		rowsByQuestion[row.QuestionID] = row
	}
	if err := scanner.Err(); err != nil {
		return locomoBackendComparisonEntry{}, fmt.Errorf("scan external %s results: %w", name, err)
	}
	if len(rowsByQuestion) == 0 {
		if statusReason == "" {
			statusReason = externalBackendNotComparableReason(name)
		}
		return locomoBackendComparisonEntry{Backend: name, Comparable: false, NotComparableReason: statusReason, SetupNotes: setupNotes}, nil
	}
	validMemoryIDs := make(map[string]struct{}, len(data.Memories))
	for _, mem := range data.Memories {
		validMemoryIDs[mem.MemoryID] = struct{}{}
	}
	results := make([]locomoQuestionResult, 0, len(data.Questions))
	for _, q := range data.Questions {
		row, ok := rowsByQuestion[q.QuestionID]
		if !ok {
			return locomoBackendComparisonEntry{}, fmt.Errorf("external %s missing question_id %q", name, q.QuestionID)
		}
		ids := make([]string, 0, len(row.Results))
		seen := map[string]struct{}{}
		for _, hit := range row.Results {
			id := strings.TrimSpace(hit.MemoryID)
			if id == "" {
				return locomoBackendComparisonEntry{}, fmt.Errorf("external %s question %s returned empty memory_id", name, q.QuestionID)
			}
			if _, exists := validMemoryIDs[id]; !exists {
				return locomoBackendComparisonEntry{}, fmt.Errorf("external %s question %s returned unknown memory_id %q", name, q.QuestionID, id)
			}
			if _, exists := seen[id]; exists {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
		results = append(results, scoreLocomoQuestion(q, ids))
	}
	summary := summarizeLocomoSystem(name, results)
	return locomoBackendComparisonEntry{Backend: name, Comparable: true, Questions: len(results), RecallAnyAt5: summary.RecallAnyAt5, RecallAnyAt10: summary.RecallAnyAt10, StrictRecallAt5: summary.StrictRecallAt5, StrictRecallAt10: summary.StrictRecallAt10, MRR: summary.MRR, FailureCategories: locomoFailureCategories(results), QuestionsDetail: results, SetupNotes: setupNotes}, nil
}

func newLocomoBackend(name string) (MemoryBackend, string, error) {
	switch name {
	case "random":
		return &randomBackend{items: map[string]backendMemory{}, byConversation: map[string][]backendMemory{}}, "", nil
	case "recency":
		return &recencyBackend{items: map[string]backendMemory{}, byConversation: map[string][]backendMemory{}}, "", nil
	case "bm25":
		return &bm25Backend{items: map[string]backendMemory{}, byConversation: map[string][]backendMemory{}}, "", nil
	case "sqlite-fts5":
		return newSQLiteFTSBackend()
	case "goncho":
		return newGonchoBackend()
	case "agentmemory":
		return nil, externalBackendNotComparableReason("agentmemory"), nil
	case "mem0":
		return nil, externalBackendNotComparableReason("mem0"), nil
	default:
		return nil, "", fmt.Errorf("unknown backend %q", name)
	}
}

func searchLocomoBackend(ctx context.Context, backend MemoryBackend, q locomoQuestionRow, topK int) ([]BackendResult, error) {
	if scoped, ok := backend.(scopedMemoryBackend); ok {
		return scoped.SearchScoped(ctx, q.ConversationID, q.Question, topK)
	}
	return backend.Search(ctx, q.Question, topK)
}

type backendMemory struct {
	ID             string
	Content        string
	ConversationID string
	Metadata       map[string]any
	Seq            int
}

type randomBackend struct {
	items          map[string]backendMemory
	byConversation map[string][]backendMemory
}

func (b *randomBackend) Name() string { return "random" }
func (b *randomBackend) Reset(context.Context) error {
	b.items = map[string]backendMemory{}
	b.byConversation = map[string][]backendMemory{}
	return nil
}
func (b *randomBackend) Insert(_ context.Context, id, content string, metadata map[string]any) error {
	item := backendMemory{ID: id, Content: content, ConversationID: metadataString(metadata, "conversation_id"), Metadata: metadata, Seq: len(b.items)}
	b.items[id] = item
	b.byConversation[item.ConversationID] = append(b.byConversation[item.ConversationID], item)
	return nil
}
func (b *randomBackend) Search(_ context.Context, question string, topK int) ([]BackendResult, error) {
	return b.searchItems(question, topK, backendSortedItems(b.items)), nil
}
func (b *randomBackend) SearchScoped(_ context.Context, conversationID, question string, topK int) ([]BackendResult, error) {
	return b.searchItems(question, topK, backendItemsForConversation(b.byConversation, conversationID)), nil
}
func (b *randomBackend) searchItems(question string, topK int, items []backendMemory) []BackendResult {
	sort.SliceStable(items, func(i, j int) bool {
		return stableHash(question+"/"+items[i].ID) < stableHash(question+"/"+items[j].ID)
	})
	return backendFirstResults(items, topK)
}
func (b *randomBackend) Close(context.Context) error { return nil }

type recencyBackend struct {
	items          map[string]backendMemory
	byConversation map[string][]backendMemory
}

func (b *recencyBackend) Name() string { return "recency" }
func (b *recencyBackend) Reset(context.Context) error {
	b.items = map[string]backendMemory{}
	b.byConversation = map[string][]backendMemory{}
	return nil
}
func (b *recencyBackend) Insert(_ context.Context, id, content string, metadata map[string]any) error {
	item := backendMemory{ID: id, Content: content, ConversationID: metadataString(metadata, "conversation_id"), Metadata: metadata, Seq: len(b.items)}
	b.items[id] = item
	b.byConversation[item.ConversationID] = append(b.byConversation[item.ConversationID], item)
	return nil
}
func (b *recencyBackend) Search(_ context.Context, _ string, topK int) ([]BackendResult, error) {
	return b.searchItems(topK, backendSortedItems(b.items)), nil
}
func (b *recencyBackend) SearchScoped(_ context.Context, conversationID, _ string, topK int) ([]BackendResult, error) {
	return b.searchItems(topK, backendItemsForConversation(b.byConversation, conversationID)), nil
}
func (b *recencyBackend) searchItems(topK int, items []backendMemory) []BackendResult {
	sort.SliceStable(items, func(i, j int) bool { return items[i].Seq > items[j].Seq })
	return backendFirstResults(items, topK)
}
func (b *recencyBackend) Close(context.Context) error { return nil }

type bm25Backend struct {
	items          map[string]backendMemory
	byConversation map[string][]backendMemory
}

func (b *bm25Backend) Name() string { return "bm25" }
func (b *bm25Backend) Reset(context.Context) error {
	b.items = map[string]backendMemory{}
	b.byConversation = map[string][]backendMemory{}
	return nil
}
func (b *bm25Backend) Insert(_ context.Context, id, content string, metadata map[string]any) error {
	item := backendMemory{ID: id, Content: content, ConversationID: metadataString(metadata, "conversation_id"), Metadata: metadata, Seq: len(b.items)}
	b.items[id] = item
	b.byConversation[item.ConversationID] = append(b.byConversation[item.ConversationID], item)
	return nil
}
func (b *bm25Backend) Search(_ context.Context, question string, topK int) ([]BackendResult, error) {
	return b.searchItems(question, topK, backendSortedItems(b.items)), nil
}
func (b *bm25Backend) SearchScoped(_ context.Context, conversationID, question string, topK int) ([]BackendResult, error) {
	return b.searchItems(question, topK, backendItemsForConversation(b.byConversation, conversationID)), nil
}
func (b *bm25Backend) searchItems(question string, topK int, items []backendMemory) []BackendResult {
	records := make([]MemoryRecord, 0, len(items))
	for _, item := range items {
		records = append(records, MemoryRecord{ID: item.ID, Peer: item.ConversationID, Content: item.Content})
	}
	ranked := rankMemoriesBM25(question, records)
	out := make([]BackendResult, 0, min(topK, len(ranked)))
	for i, item := range ranked[:min(topK, len(ranked))] {
		out = append(out, BackendResult{MemoryID: item.ID, Score: float64(topK - i)})
	}
	return out
}
func (b *bm25Backend) Close(context.Context) error { return nil }

type sqliteFTSBackend struct {
	dir   string
	store *memory.SqliteStore
	db    *sql.DB
}

func newSQLiteFTSBackend() (*sqliteFTSBackend, string, error) {
	dir, err := os.MkdirTemp("", "locomo-backend-fts-*")
	if err != nil {
		return nil, "", err
	}
	return &sqliteFTSBackend{dir: dir}, "", nil
}
func (b *sqliteFTSBackend) Name() string { return "sqlite-fts5" }
func (b *sqliteFTSBackend) Reset(ctx context.Context) error {
	store, err := memory.OpenSqlite(filepath.Join(b.dir, "fts.db"), 0, nil)
	if err != nil {
		return err
	}
	b.store, b.db = store, store.DB()
	_, err = b.db.ExecContext(ctx, `CREATE VIRTUAL TABLE locomo_fts USING fts5(id UNINDEXED, conversation_id UNINDEXED, content)`)
	return err
}
func (b *sqliteFTSBackend) Insert(ctx context.Context, id, content string, metadata map[string]any) error {
	_, err := b.db.ExecContext(ctx, `INSERT INTO locomo_fts(id, conversation_id, content) VALUES(?, ?, ?)`, id, metadataString(metadata, "conversation_id"), content)
	return err
}
func (b *sqliteFTSBackend) Search(ctx context.Context, question string, topK int) ([]BackendResult, error) {
	query := ftsQuery(question)
	if query == "" {
		return nil, nil
	}
	rows, err := b.db.QueryContext(ctx, `SELECT id, bm25(locomo_fts) FROM locomo_fts WHERE locomo_fts MATCH ? ORDER BY bm25(locomo_fts) LIMIT ?`, query, topK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BackendResult
	for rows.Next() {
		var r BackendResult
		if err := rows.Scan(&r.MemoryID, &r.Score); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
func (b *sqliteFTSBackend) SearchScoped(ctx context.Context, conversationID, question string, topK int) ([]BackendResult, error) {
	query := ftsQuery(question)
	if query == "" {
		return nil, nil
	}
	rows, err := b.db.QueryContext(ctx, `SELECT id, bm25(locomo_fts) FROM locomo_fts WHERE conversation_id = ? AND locomo_fts MATCH ? ORDER BY bm25(locomo_fts) LIMIT ?`, conversationID, query, topK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BackendResult
	for rows.Next() {
		var r BackendResult
		if err := rows.Scan(&r.MemoryID, &r.Score); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (b *sqliteFTSBackend) Close(ctx context.Context) error {
	if b.store != nil {
		_ = b.store.Close(ctx)
	}
	if b.dir != "" {
		return os.RemoveAll(b.dir)
	}
	return nil
}

type gonchoBackend struct {
	dir        string
	store      *memory.SqliteStore
	svc        *goncho.Service
	contentIDs map[string][]string
}

func newGonchoBackend() (*gonchoBackend, string, error) {
	dir, err := os.MkdirTemp("", "locomo-backend-goncho-*")
	if err != nil {
		return nil, "", err
	}
	return &gonchoBackend{dir: dir, contentIDs: map[string][]string{}}, "", nil
}
func (b *gonchoBackend) Name() string { return "goncho" }
func (b *gonchoBackend) Reset(ctx context.Context) error {
	store, err := memory.OpenSqlite(filepath.Join(b.dir, "goncho.db"), 0, nil)
	if err != nil {
		return err
	}
	b.store = store
	if err := goncho.RunMigrations(store.DB()); err != nil {
		return err
	}
	b.svc = goncho.NewService(store.DB(), goncho.Config{WorkspaceID: "locomo-backend", ObserverPeerID: "locomo-backend", RecentMessages: 0}, nil)
	return nil
}
func (b *gonchoBackend) Insert(ctx context.Context, id, content string, metadata map[string]any) error {
	peer := metadataString(metadata, "conversation_id")
	if peer == "" {
		peer = "locomo"
	}
	result, err := b.svc.Conclude(ctx, goncho.ConcludeParams{Peer: peer, Conclusion: content, Scope: "benchmark"})
	if err != nil {
		return err
	}
	if _, err := b.store.DB().ExecContext(ctx, `UPDATE goncho_conclusions SET created_at = ?, updated_at = ? WHERE id = ?`, len(b.contentIDs)+1, len(b.contentIDs)+1, result.ID); err != nil {
		return err
	}
	b.contentIDs[contentIDKey(peer, content)] = append(b.contentIDs[contentIDKey(peer, content)], id)
	return nil
}
func (b *gonchoBackend) Search(ctx context.Context, question string, topK int) ([]BackendResult, error) {
	result, err := b.svc.Search(ctx, goncho.SearchParams{Peer: "locomo", Query: question, Limit: topK, MaxTokens: 100_000})
	if err != nil {
		return nil, err
	}
	var out []BackendResult
	seen := map[string]struct{}{}
	for rank, hit := range result.Results {
		for _, id := range b.contentIDs[contentIDKey("locomo", hit.Content)] {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, BackendResult{MemoryID: id, Score: float64(topK - rank)})
		}
	}
	return out, nil
}
func (b *gonchoBackend) SearchScoped(ctx context.Context, conversationID, question string, topK int) ([]BackendResult, error) {
	result, err := b.svc.Search(ctx, goncho.SearchParams{Peer: conversationID, Query: question, Limit: topK, MaxTokens: 100_000})
	if err != nil {
		return nil, err
	}
	var out []BackendResult
	seen := map[string]struct{}{}
	for rank, hit := range result.Results {
		for _, id := range b.contentIDs[contentIDKey(conversationID, hit.Content)] {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, BackendResult{MemoryID: id, Score: float64(topK - rank)})
		}
	}
	return out, nil
}
func (b *gonchoBackend) Close(ctx context.Context) error {
	if b.store != nil {
		_ = b.store.Close(ctx)
	}
	if b.dir != "" {
		return os.RemoveAll(b.dir)
	}
	return nil
}

func locomoMemoryMetadata(mem locomoMemoryRow) map[string]any {
	return map[string]any{"conversation_id": mem.ConversationID, "session_id": mem.SessionID, "speaker": mem.Speaker, "turn_index": mem.TurnIndex, "timestamp": mem.Timestamp}
}
func metadataString(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	if value, ok := metadata[key].(string); ok {
		return value
	}
	return ""
}
func backendItemsForConversation(index map[string][]backendMemory, conversationID string) []backendMemory {
	out := append([]backendMemory(nil), index[conversationID]...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
func backendSortedItems(m map[string]backendMemory) []backendMemory {
	out := make([]backendMemory, 0, len(m))
	for _, item := range m {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
func backendFirstResults(items []backendMemory, topK int) []BackendResult {
	if topK > len(items) {
		topK = len(items)
	}
	out := make([]BackendResult, 0, topK)
	for i, item := range items[:topK] {
		out = append(out, BackendResult{MemoryID: item.ID, Score: float64(topK - i)})
	}
	return out
}
func locomoFailureCategories(results []locomoQuestionResult) map[string]int {
	out := map[string]int{}
	for _, q := range results {
		if q.Rank == 0 {
			out["miss_top_10"]++
		} else if q.Rank > 1 {
			out["gold_not_rank_1"]++
		} else {
			out["gold_rank_1"]++
		}
	}
	return out
}
func setupNotesForBackend(name string) []string {
	switch name {
	case "agentmemory":
		return []string{"External Python probe: scripts/bench_agentmemory_locomo.py --capability.", "Comparable when AGENTMEMORY_SOURCE_DIR points at PR #583 / commit 9b18a80c9d2839b025279978d3f4b5e1f9bc6e74 with npm dependencies installed.", "Adapter path uses standalone InMemoryKV fallback: memory_save external_id plus metadata.memory_id, then memory_smart_search. This validates stable IDs but is not the full running agentmemory server.", "If AGENTMEMORY_SOURCE_DIR is absent, agentmemory is marked not comparable."}
	case "mem0":
		return []string{"External Python probe: scripts/bench_mem0_locomo.py --capability.", "Exact package version used in this run: none; backend marked not comparable before scoring.", "Install candidate: pip install mem0ai, with local/vector dependencies configured by upstream mem0 docs.", "Current status: not comparable in this harness until search results return caller-supplied memory_id unchanged without LLM answer scoring."}
	default:
		return []string{"Local deterministic adapter in cmd/goncho-bench."}
	}
}
func externalBackendNotComparableReason(name string) string {
	switch name {
	case "agentmemory":
		return "not comparable: no stable-memory-id LOCOMO adapter is wired for agentmemory; scoring requires search results to return the inserted memory_id exactly"
	case "mem0":
		return "not comparable: no stable-memory-id LOCOMO adapter is wired for mem0; scoring requires search results to return the inserted memory_id exactly"
	default:
		return "not comparable: stable memory IDs unavailable"
	}
}
func currentRSSBytes() uint64 { var m runtime.MemStats; runtime.ReadMemStats(&m); return m.Sys }

func writeLocomoBackendComparisonJSON(path string, report locomoBackendComparisonReport) error {
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

func writeLocomoBackendComparisonFailures(path string, data locomoDataset, report locomoBackendComparisonReport) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	memByID := map[string]locomoMemoryRow{}
	for _, mem := range data.Memories {
		memByID[mem.MemoryID] = mem
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(file)
	for _, backend := range report.Backends {
		if !backend.Comparable {
			row := map[string]any{"backend": backend.Backend, "comparable": false, "failure_category": "not_comparable", "notes": backend.NotComparableReason}
			if err := enc.Encode(row); err != nil {
				_ = file.Close()
				return err
			}
			continue
		}
		for _, q := range backend.QuestionsDetail {
			if q.Rank == 1 {
				continue
			}
			row := locomoFailureRow{QuestionID: q.QuestionID, Category: q.Category, FailureCategory: backend.Backend + ":" + locomoFailureNotes(q), Question: q.Question, GoldMemoryIDs: q.GoldMemoryIDs, Notes: locomoFailureNotes(q)}
			for i, id := range q.RetrievedIDs[:min(10, len(q.RetrievedIDs))] {
				mem := memByID[id]
				row.TopHits = append(row.TopHits, locomoFailureHit{Rank: i + 1, MemoryID: id, Content: mem.Content, Score: 0, Speaker: mem.Speaker, SessionID: mem.SessionID, TurnIndex: mem.TurnIndex})
			}
			wrapped := map[string]any{"backend": backend.Backend, "comparable": true, "failure": row}
			if err := enc.Encode(wrapped); err != nil {
				_ = file.Close()
				return err
			}
		}
	}
	return file.Close()
}

func writeLocomoBackendComparisonMarkdown(path string, report locomoBackendComparisonReport, jsonPath, failuresPath string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("# LOCOMO External Backend Comparison\n\n")
	b.WriteString("This is a benchmark adapter suite, not a marketing dunk. It compares retrieval backends only when they can return stable inserted memory IDs.\n\n")
	if strings.TrimSpace(jsonPath) != "" {
		fmt.Fprintf(&b, "- JSON evidence: `%s`\n", jsonPath)
	}
	if strings.TrimSpace(failuresPath) != "" {
		fmt.Fprintf(&b, "- Failure JSONL: `%s`\n", failuresPath)
	}
	fmt.Fprintf(&b, "- Memories: `%s`\n- Questions: `%s`\n- Questions: `%d`\n- Memories: `%d`\n- no_llm_judge: `%t`\n\n", report.FixturePaths.Memories, report.FixturePaths.Questions, report.QuestionCount, report.MemoryCount, report.NoLLMJudge)
	b.WriteString("## Rules\n\n")
	for _, rule := range report.Rules {
		fmt.Fprintf(&b, "- %s\n", rule)
	}
	b.WriteString("\n## Results\n\n| Backend | Comparable | recall_any@5 | recall_any@10 | strict@5 | strict@10 | MRR | Search latency ms | Notes |\n| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |\n")
	for _, e := range report.Backends {
		note := e.NotComparableReason
		if note == "" && len(e.SetupNotes) > 0 {
			note = strings.Join(e.SetupNotes, " ")
		}
		fmt.Fprintf(&b, "| `%s` | %t | %.2f%% | %.2f%% | %.2f%% | %.2f%% | %.2f%% | %d | %s |\n", e.Backend, e.Comparable, e.RecallAnyAt5*100, e.RecallAnyAt10*100, e.StrictRecallAt5*100, e.StrictRecallAt10*100, e.MRR*100, e.SearchLatencyMs, strings.ReplaceAll(note, "|", "/"))
	}
	b.WriteString("\n## Setup notes\n\n")
	b.WriteString("- Goncho, BM25, and SQLite FTS5 are local Go adapters with no hosted dependency.\n")
	b.WriteString("- agentmemory probe: `python3 scripts/bench_agentmemory_locomo.py --capability`. Comparable when `AGENTMEMORY_SOURCE_DIR` points at PR #583 / commit `9b18a80c9d2839b025279978d3f4b5e1f9bc6e74` with npm dependencies installed. This adapter uses the standalone InMemoryKV fallback, not the full running agentmemory server.\n")
	b.WriteString("- mem0 probe: `python3 scripts/bench_mem0_locomo.py --capability`. Exact package version used here: none; backend is marked not comparable before scoring. Candidate install: `pip install mem0ai` plus upstream local vector-store dependencies. Comparable only after configured local retrieval can return caller-supplied `memory_id` without answer-generation scoring.\n")
	b.WriteString("\n## Interpretation\n\nBackends marked not comparable are excluded from score claims until they implement the `MemoryBackend` contract and return the same stable `memory_id` values that were inserted. This keeps the arena fair and prevents answer-generation or LLM-judge effects from leaking into retrieval metrics.\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

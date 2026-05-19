package goncho

import (
	"context"
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	memory "github.com/TrebuchetDynamics/goncho/memory"
	toolmeta "github.com/TrebuchetDynamics/goncho/toolmeta"
)

type mockMemoryToolStore struct {
	mu      sync.Mutex
	entries map[string]MemoryToolEntry
}

func newMockToolStore() *mockMemoryToolStore {
	return &mockMemoryToolStore{entries: make(map[string]MemoryToolEntry)}
}

func (m *mockMemoryToolStore) Store(ctx context.Context, entry MemoryToolEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries[entry.ID] = entry
	return nil
}

func (m *mockMemoryToolStore) Retrieve(ctx context.Context, query string, limit int) ([]MemoryToolEntry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var results []MemoryToolEntry
	for _, e := range m.entries {
		if query == "" || containsTag(e.Tags, query) || containsMemoryContent(e.Content, query) {
			results = append(results, e)
			if len(results) >= limit {
				break
			}
		}
	}
	return results, nil
}

func (m *mockMemoryToolStore) Update(ctx context.Context, id string, content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, ok := m.entries[id]
	if !ok {
		return nil
	}
	entry.Content = content
	m.entries[id] = entry
	return nil
}

func (m *mockMemoryToolStore) UpdateImportance(ctx context.Context, id string, importance float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, ok := m.entries[id]
	if !ok {
		return nil
	}
	entry.Importance = importance
	m.entries[id] = entry
	return nil
}

func (m *mockMemoryToolStore) Forget(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.entries, id)
	return nil
}

func containsTag(tags []string, query string) bool {
	for _, t := range tags {
		if t == query {
			return true
		}
	}
	return false
}

func containsMemoryContent(content string, query string) bool {
	content = strings.ToLower(content)
	query = strings.ToLower(query)
	return strings.Contains(content, query) || strings.Contains(query, content)
}

func TestStoreMemory(t *testing.T) {
	store := newMockToolStore()
	tool := &storeMemoryTool{newMemoryToolBase(store)}
	args := json.RawMessage(`{"content":"test memory","tags":["test"],"importance":0.8}`)
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("store_memory failed: %v", err)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(result, &out); err != nil {
		t.Fatalf("invalid result json: %v", err)
	}
	if out["success"] != true {
		t.Fatal("store_memory did not succeed")
	}
}

func TestStoreMemory_MissingContent(t *testing.T) {
	store := newMockToolStore()
	tool := &storeMemoryTool{newMemoryToolBase(store)}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"tags":["test"]}`))
	if err == nil {
		t.Fatal("store_memory should fail with missing content")
	}
}

func TestRetrieveMemory(t *testing.T) {
	store := newMockToolStore()
	store.Store(context.Background(), MemoryToolEntry{ID: "mem_1", Content: "hello world", Tags: []string{"greeting"}, Importance: 0.9})
	tool := &retrieveMemoryTool{newMemoryToolBase(store)}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{"query":"greeting","limit":5}`))
	if err != nil {
		t.Fatalf("retrieve_memory failed: %v", err)
	}
	var out map[string]interface{}
	json.Unmarshal(result, &out)
	results, ok := out["results"].([]interface{})
	if !ok || len(results) == 0 {
		t.Fatal("retrieve_memory did not return results")
	}
}

func TestUpdateMemory(t *testing.T) {
	store := newMockToolStore()
	store.Store(context.Background(), MemoryToolEntry{ID: "mem_1", Content: "old content"})
	tool := &updateMemoryTool{newMemoryToolBase(store)}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{"id":"mem_1","content":"new content"}`))
	if err != nil {
		t.Fatalf("update_memory failed: %v", err)
	}
	var out map[string]interface{}
	json.Unmarshal(result, &out)
	if out["success"] != true {
		t.Fatal("update_memory did not succeed")
	}
}

func TestUpdateMemoryImportance(t *testing.T) {
	store := newMockToolStore()
	store.Store(context.Background(), MemoryToolEntry{ID: "mem_1", Content: "old content", Importance: 0.2})
	tool := &updateMemoryTool{newMemoryToolBase(store)}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{"id":"mem_1","importance":0.95}`))
	if err != nil {
		t.Fatalf("update_memory importance failed: %v", err)
	}
	var out map[string]interface{}
	json.Unmarshal(result, &out)
	if out["success"] != true {
		t.Fatal("update_memory importance did not succeed")
	}
	if got := store.entries["mem_1"].Importance; got != 0.95 {
		t.Fatalf("importance = %v, want 0.95", got)
	}
}

func TestSummarizeMemories(t *testing.T) {
	store := newMockToolStore()
	store.Store(context.Background(), MemoryToolEntry{ID: "m1", Content: "a", Tags: []string{"proj"}})
	store.Store(context.Background(), MemoryToolEntry{ID: "m2", Content: "b", Tags: []string{"proj"}})
	tool := &summarizeMemoryTool{newMemoryToolBase(store)}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{"filter":"proj","max_items":5}`))
	if err != nil {
		t.Fatalf("summarize_memories failed: %v", err)
	}
	var out map[string]interface{}
	json.Unmarshal(result, &out)
	if out["summarized"].(float64) == 0 {
		t.Fatal("summarize_memories did not return summarized count")
	}
}

func TestForgetMemory(t *testing.T) {
	store := newMockToolStore()
	store.Store(context.Background(), MemoryToolEntry{ID: "mem_1", Content: "to forget"})
	tool := &forgetMemoryTool{newMemoryToolBase(store)}
	result, err := tool.Execute(context.Background(), json.RawMessage(`{"id":"mem_1"}`))
	if err != nil {
		t.Fatalf("forget_memory failed: %v", err)
	}
	var out map[string]interface{}
	json.Unmarshal(result, &out)
	if out["success"] != true {
		t.Fatal("forget_memory did not succeed")
	}
}

func TestMemoryToolNames(t *testing.T) {
	store := newMockToolStore()
	tests := []struct {
		want string
		tool interface{ Name() string }
	}{
		{"store_memory", &storeMemoryTool{newMemoryToolBase(store)}},
		{"retrieve_memory", &retrieveMemoryTool{newMemoryToolBase(store)}},
		{"update_memory", &updateMemoryTool{newMemoryToolBase(store)}},
		{"summarize_memories", &summarizeMemoryTool{newMemoryToolBase(store)}},
		{"forget_memory", &forgetMemoryTool{newMemoryToolBase(store)}},
	}
	for _, tc := range tests {
		if tc.tool.Name() != tc.want {
			t.Errorf("tool Name() = %q, want %q", tc.tool.Name(), tc.want)
		}
	}
}

func TestMemoryToolsExposeOperationSpecs(t *testing.T) {
	store := newMockToolStore()
	tests := []struct {
		tool       toolmeta.Tool
		mutating   bool
		idempotent bool
	}{
		{NewStoreMemoryTool(store), true, false},
		{NewRetrieveMemoryTool(store), false, true},
		{NewUpdateMemoryTool(store), true, false},
		{NewSummarizeMemoryTool(store), false, true},
		{NewForgetMemoryTool(store), true, true},
	}
	for _, tc := range tests {
		specTool, ok := tc.tool.(toolmeta.Spec)
		if !ok {
			t.Fatalf("%s does not expose tools.OperationSpec", tc.tool.Name())
		}
		spec := specTool.Spec()
		if spec.Name != tc.tool.Name() || spec.Description != tc.tool.Description() || string(spec.Schema) != string(tc.tool.Schema()) {
			t.Fatalf("%s spec descriptor = %+v, want live tool descriptor", tc.tool.Name(), spec.ToolDescriptor)
		}
		if spec.AuditKind != "memory" || !spec.PromptSafe {
			t.Fatalf("%s spec = %+v, want prompt-safe memory audit spec", tc.tool.Name(), spec)
		}
		if !stringSliceContains(spec.TrustClass, "operator") || !stringSliceContains(spec.TrustClass, "system") {
			t.Fatalf("%s trust class = %#v, want operator and system", tc.tool.Name(), spec.TrustClass)
		}
		if spec.Mutating != tc.mutating || spec.Idempotent != tc.idempotent {
			t.Fatalf("%s mutating/idempotent = %v/%v, want %v/%v", tc.tool.Name(), spec.Mutating, spec.Idempotent, tc.mutating, tc.idempotent)
		}
	}
}

func TestMemoryToolsStoreRetrieveUpdateSummarizeForgetWithMetadata(t *testing.T) {
	ctx := context.Background()
	sqlite, store := newLocalMarkdownToolStore(t, ctx)

	storeTool := NewStoreMemoryTool(store)
	retrieveTool := NewRetrieveMemoryTool(store)
	updateTool := NewUpdateMemoryTool(store)
	summarizeTool := NewSummarizeMemoryTool(store)
	forgetTool := NewForgetMemoryTool(store)

	stored := executeMemoryTool(t, ctx, storeTool, `{
		"content":"Goncho memory should preserve latency budget lessons for Telegram replies.",
		"tags":["goncho","latency"],
		"importance":0.95,
		"metadata":{"origin":"tdd","session":"telegram:42"}
	}`)
	id := stringField(t, stored, "id")
	if id == "" {
		t.Fatal("store_memory returned empty id")
	}

	var agentID, workspaceID, peerID, sessionKey, tagsRaw, provenanceRaw string
	var active int
	var importance float64
	if err := sqlite.DB().QueryRowContext(ctx, `
		SELECT agent_id, workspace_id, peer_id, session_key, tags_json, importance, active, provenance_json
		FROM goncho_memory_items
		WHERE memory_id = ?
	`, id).Scan(&agentID, &workspaceID, &peerID, &sessionKey, &tagsRaw, &importance, &active, &provenanceRaw); err != nil {
		t.Fatalf("read stored memory row: %v", err)
	}
	if agentID != "agent-a" || workspaceID != "workspace-a" || peerID != "user-a" || sessionKey != "telegram:42" || active != 1 {
		t.Fatalf("stored scope = agent:%q workspace:%q peer:%q session:%q active:%d", agentID, workspaceID, peerID, sessionKey, active)
	}
	if importance != 0.95 || !strings.Contains(tagsRaw, "latency") || !strings.Contains(provenanceRaw, `"origin":"tdd"`) {
		t.Fatalf("stored metadata tags/provenance = importance:%v tags:%s provenance:%s", importance, tagsRaw, provenanceRaw)
	}

	retrieved := executeMemoryTool(t, ctx, retrieveTool, `{"query":"latency","limit":5}`)
	results := memoryResults(t, retrieved)
	if len(results) != 1 || results[0].ID != id || !strings.Contains(results[0].Content, "latency budget lessons") {
		t.Fatalf("retrieve results = %+v, want stored latency memory", results)
	}

	updatedContent := "Goncho memory should keep Telegram latency under eighty milliseconds using local retrieval."
	updated := executeMemoryTool(t, ctx, updateTool, `{"id":"`+id+`","content":"`+updatedContent+`"}`)
	if updated["success"] != true {
		t.Fatalf("update result = %+v, want success", updated)
	}
	retrieved = executeMemoryTool(t, ctx, retrieveTool, `{"query":"eighty milliseconds","limit":5}`)
	results = memoryResults(t, retrieved)
	if len(results) != 1 || results[0].ID != id || results[0].Content != updatedContent {
		t.Fatalf("updated retrieve results = %+v", results)
	}

	demoted := executeMemoryTool(t, ctx, updateTool, `{"id":"`+id+`","importance":0.2}`)
	if demoted["success"] != true {
		t.Fatalf("importance update result = %+v, want success", demoted)
	}
	if err := sqlite.DB().QueryRowContext(ctx, `
		SELECT importance
		FROM goncho_memory_items
		WHERE memory_id = ?
	`, id).Scan(&importance); err != nil {
		t.Fatalf("read updated importance: %v", err)
	}
	if importance != 0.2 {
		t.Fatalf("updated importance = %v, want 0.2", importance)
	}

	summary := executeMemoryTool(t, ctx, summarizeTool, `{"filter":"latency","max_items":10}`)
	summaryText := stringField(t, summary, "summary")
	if !strings.Contains(summaryText, "eighty milliseconds") || !strings.Contains(summaryText, id) {
		t.Fatalf("summary = %q, want compressed updated memory with source id", summaryText)
	}

	chained := executeMemoryTool(t, ctx, storeTool, jsonArgs(t, map[string]any{
		"content":    summaryText,
		"tags":       []string{"goncho", "summary"},
		"importance": 0.7,
		"metadata": map[string]string{
			"origin":        "summarize_memories",
			"source_memory": id,
		},
	}))
	summaryID := stringField(t, chained, "id")
	chainedRetrieve := executeMemoryTool(t, ctx, retrieveTool, `{"query":"`+id+`","limit":5}`)
	chainedResults := memoryResults(t, chainedRetrieve)
	if len(chainedResults) != 1 || chainedResults[0].ID != summaryID || !strings.Contains(chainedResults[0].Content, id) {
		t.Fatalf("chained retrieve results = %+v, want stored summary", chainedResults)
	}

	forgotten := executeMemoryTool(t, ctx, forgetTool, `{"id":"`+id+`"}`)
	if forgotten["success"] != true {
		t.Fatalf("forget result = %+v, want success", forgotten)
	}
	retrievedAfterForget := executeMemoryTool(t, ctx, retrieveTool, `{"query":"eighty milliseconds","limit":5}`)
	for _, result := range memoryResults(t, retrievedAfterForget) {
		if result.ID == id {
			t.Fatalf("forgotten memory remained active in results: %+v", result)
		}
	}
}

func TestMemoryToolsKeepAgentMemoryIndependentInSharedStore(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	sqlite, err := memory.OpenSqlite(filepath.Join(dir, "memory.db"), 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlite.Close(ctx); err != nil {
			t.Fatalf("Close sqlite: %v", err)
		}
	})
	agentA := NewLocalMarkdownMemoryStore(sqlite.DB(), LocalMarkdownMemoryConfig{
		Path:        filepath.Join(dir, "agent-a.md"),
		AgentID:     "agent-a",
		WorkspaceID: "workspace-a",
		PeerID:      "user-a",
		SessionID:   "session-a",
	})
	agentB := NewLocalMarkdownMemoryStore(sqlite.DB(), LocalMarkdownMemoryConfig{
		Path:        filepath.Join(dir, "agent-b.md"),
		AgentID:     "agent-b",
		WorkspaceID: "workspace-b",
		PeerID:      "user-b",
		SessionID:   "session-b",
	})

	storedA := executeMemoryTool(t, ctx, NewStoreMemoryTool(agentA), `{
		"content":"Agent A private latency strategy must not cross agents.",
		"tags":["private","agent-a"],
		"importance":0.9
	}`)
	idA := stringField(t, storedA, "id")

	bView := executeMemoryTool(t, ctx, NewRetrieveMemoryTool(agentB), `{"query":"private latency strategy","limit":5}`)
	if results := memoryResults(t, bView); len(results) != 0 {
		t.Fatalf("agent B retrieved agent A memory: %+v", results)
	}

	_ = executeMemoryTool(t, ctx, NewUpdateMemoryTool(agentB), `{"id":"`+idA+`","content":"Agent B overwrite attempt."}`)
	aView := executeMemoryTool(t, ctx, NewRetrieveMemoryTool(agentA), `{"query":"private","limit":5}`)
	aResults := memoryResults(t, aView)
	if len(aResults) != 1 || aResults[0].ID != idA || strings.Contains(aResults[0].Content, "overwrite") {
		t.Fatalf("agent B update changed agent A memory: %+v", aResults)
	}

	_ = executeMemoryTool(t, ctx, NewForgetMemoryTool(agentB), `{"id":"`+idA+`"}`)
	aView = executeMemoryTool(t, ctx, NewRetrieveMemoryTool(agentA), `{"query":"private","limit":5}`)
	aResults = memoryResults(t, aView)
	if len(aResults) != 1 || aResults[0].ID != idA {
		t.Fatalf("agent B forget removed agent A memory: %+v", aResults)
	}
}

func TestImportanceScorer_LocalMarkdownRetrieveRanksRelevanceImportanceAndRecency(t *testing.T) {
	ctx := context.Background()
	sqlite, store := newLocalMarkdownToolStore(t, ctx)
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	entries := []MemoryToolEntry{
		{ID: "old-high", Content: "Telegram latency budget from an old incident.", Tags: []string{"latency"}, Importance: 0.95, CreatedAt: now.Add(-90 * 24 * time.Hour), UpdatedAt: now.Add(-90 * 24 * time.Hour)},
		{ID: "fresh-low", Content: "Telegram latency note from this morning.", Tags: []string{"latency"}, Importance: 0.2, CreatedAt: now.Add(-1 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)},
		{ID: "fresh-important", Content: "Telegram latency SLO must stay below eighty milliseconds.", Tags: []string{"latency", "slo"}, Importance: 0.8, CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)},
		{ID: "irrelevant", Content: "Theme preference is dark.", Tags: []string{"theme"}, Importance: 1.0, CreatedAt: now, UpdatedAt: now},
	}
	for _, entry := range entries {
		if err := store.Store(ctx, entry); err != nil {
			t.Fatalf("store %s: %v", entry.ID, err)
		}
	}

	results, err := store.Retrieve(ctx, "Telegram latency", 4)
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	var got []string
	for _, result := range results {
		got = append(got, result.ID)
	}
	want := []string{"fresh-important", "fresh-low", "old-high"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("retrieve ranked IDs = %v, want %v", got, want)
	}

	var irrelevantActive int
	if err := sqlite.DB().QueryRowContext(ctx, `SELECT active FROM goncho_memory_items WHERE memory_id = 'irrelevant'`).Scan(&irrelevantActive); err != nil {
		t.Fatalf("read irrelevant row: %v", err)
	}
	if irrelevantActive != 1 {
		t.Fatalf("irrelevant memory active = %d, want retained but not returned", irrelevantActive)
	}
}

func newLocalMarkdownToolStore(t *testing.T, ctx context.Context) (*memory.SqliteStore, *LocalMarkdownMemoryStore) {
	t.Helper()
	dir := t.TempDir()
	sqlite, err := memory.OpenSqlite(filepath.Join(dir, "memory.db"), 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlite.Close(ctx); err != nil {
			t.Fatalf("Close sqlite: %v", err)
		}
	})
	return sqlite, NewLocalMarkdownMemoryStore(sqlite.DB(), LocalMarkdownMemoryConfig{
		Path:           filepath.Join(dir, "GONCHO_MEMORY.md"),
		AgentID:        "agent-a",
		WorkspaceID:    "workspace-a",
		ObserverPeerID: "agent-a",
		PeerID:         "user-a",
		SessionID:      "telegram:42",
	})
}

func executeMemoryTool(t *testing.T, ctx context.Context, tool toolmeta.Tool, args string) map[string]any {
	t.Helper()
	raw, err := tool.Execute(ctx, json.RawMessage(args))
	if err != nil {
		t.Fatalf("%s Execute: %v", tool.Name(), err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode %s output %s: %v", tool.Name(), raw, err)
	}
	return out
}

func stringField(t *testing.T, out map[string]any, name string) string {
	t.Helper()
	value, ok := out[name].(string)
	if !ok {
		t.Fatalf("%s = %#v, want string", name, out[name])
	}
	return value
}

func memoryResults(t *testing.T, out map[string]any) []MemoryToolEntry {
	t.Helper()
	raw, err := json.Marshal(out["results"])
	if err != nil {
		t.Fatalf("marshal results: %v", err)
	}
	var results []MemoryToolEntry
	if err := json.Unmarshal(raw, &results); err != nil {
		t.Fatalf("decode results: %v", err)
	}
	return results
}

func jsonArgs(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	return string(raw)
}

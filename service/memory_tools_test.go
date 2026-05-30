package goncho

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	memory "github.com/TrebuchetDynamics/goncho/memory"
	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
	"github.com/TrebuchetDynamics/goncho/toolmeta"
)

func TestMemoryToolsPublicFacadeExposeOperationSpecs(t *testing.T) {
	ctx := context.Background()
	_, store := newLocalMarkdownToolStore(t, ctx)
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
		if !sliceutil.Contains(spec.TrustClass, "operator") || !sliceutil.Contains(spec.TrustClass, "system") {
			t.Fatalf("%s trust class = %#v, want operator and system", tc.tool.Name(), spec.TrustClass)
		}
		if spec.Mutating != tc.mutating || spec.Idempotent != tc.idempotent {
			t.Fatalf("%s mutating/idempotent = %v/%v, want %v/%v", tc.tool.Name(), spec.Mutating, spec.Idempotent, tc.mutating, tc.idempotent)
		}
	}
}

func TestMemoryToolsPublicFacadeStoreRetrieveUpdateSummarizeForgetWithMetadata(t *testing.T) {
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

func TestMemoryToolsPublicFacadeKeepAgentMemoryIndependentInSharedStore(t *testing.T) {
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

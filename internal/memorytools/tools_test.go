package memorytools

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/TrebuchetDynamics/goncho/toolmeta"
)

type mockStore struct {
	mu      sync.Mutex
	entries map[string]Entry
}

func newMockStore() *mockStore {
	return &mockStore{entries: make(map[string]Entry)}
}

func (m *mockStore) Store(ctx context.Context, entry Entry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries[entry.ID] = entry
	return nil
}

func (m *mockStore) Retrieve(ctx context.Context, query string, limit int) ([]Entry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var results []Entry
	for _, entry := range m.entries {
		if query == "" || containsTag(entry.Tags, query) || containsContent(entry.Content, query) {
			results = append(results, entry)
			if len(results) >= limit {
				break
			}
		}
	}
	return results, nil
}

func (m *mockStore) Update(ctx context.Context, id string, content string) error {
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

func (m *mockStore) UpdateImportance(ctx context.Context, id string, importance float64) error {
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

func (m *mockStore) Forget(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.entries, id)
	return nil
}

func TestMemoryToolsStoreRetrieveUpdateSummarizeAndForget(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()

	stored := executeTool(t, ctx, NewStoreTool(store), `{"content":"test memory","tags":["test"],"importance":1.2}`)
	id := stringField(t, stored, "id")
	if id == "" || stored["success"] != true {
		t.Fatalf("store result = %+v, want success with id", stored)
	}
	if got := store.entries[id].Importance; got != 1 {
		t.Fatalf("stored importance = %v, want clamped 1", got)
	}

	retrieved := executeTool(t, ctx, NewRetrieveTool(store), `{"query":"test","limit":5}`)
	if results := entriesField(t, retrieved, "results"); len(results) != 1 || results[0].ID != id {
		t.Fatalf("retrieve results = %+v, want stored memory", results)
	}

	updated := executeTool(t, ctx, NewUpdateTool(store), `{"id":"`+id+`","content":"new content","importance":-0.5}`)
	if updated["success"] != true {
		t.Fatalf("update result = %+v, want success", updated)
	}
	if got := store.entries[id]; got.Content != "new content" || got.Importance != 0 {
		t.Fatalf("updated entry = %+v, want content change and clamped importance", got)
	}

	summary := executeTool(t, ctx, NewSummarizeTool(store), `{"filter":"new","max_items":5}`)
	if text := stringField(t, summary, "summary"); !strings.Contains(text, id) || !strings.Contains(text, "new content") {
		t.Fatalf("summary = %q, want id and content", text)
	}

	forgotten := executeTool(t, ctx, NewForgetTool(store), `{"id":"`+id+`"}`)
	if forgotten["success"] != true {
		t.Fatalf("forget result = %+v, want success", forgotten)
	}
	if _, ok := store.entries[id]; ok {
		t.Fatalf("forgotten memory %s remained in store", id)
	}
}

func TestMemoryToolsValidateRequiredInputs(t *testing.T) {
	ctx := context.Background()
	store := newMockStore()
	tests := []struct {
		name string
		tool toolmeta.Tool
		args string
		want string
	}{
		{name: "store", tool: NewStoreTool(store), args: `{"tags":["test"]}`, want: "content is required"},
		{name: "retrieve", tool: NewRetrieveTool(store), args: `{"limit":1}`, want: "query is required"},
		{name: "update id", tool: NewUpdateTool(store), args: `{"content":"new"}`, want: "id is required"},
		{name: "update body", tool: NewUpdateTool(store), args: `{"id":"mem_1"}`, want: "content or importance is required"},
		{name: "summarize", tool: NewSummarizeTool(store), args: `{"max_items":1}`, want: "filter is required"},
		{name: "forget", tool: NewForgetTool(store), args: `{}`, want: "id is required"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.tool.Execute(ctx, json.RawMessage(tc.args))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestMemoryToolsExposeOperationSpecs(t *testing.T) {
	store := newMockStore()
	tests := []struct {
		tool       toolmeta.Tool
		mutating   bool
		idempotent bool
	}{
		{NewStoreTool(store), true, false},
		{NewRetrieveTool(store), false, true},
		{NewUpdateTool(store), true, false},
		{NewSummarizeTool(store), false, true},
		{NewForgetTool(store), true, true},
	}
	for _, tc := range tests {
		specTool, ok := tc.tool.(toolmeta.Spec)
		if !ok {
			t.Fatalf("%s does not expose OperationSpec", tc.tool.Name())
		}
		spec := specTool.Spec()
		if spec.Name != tc.tool.Name() || spec.Description != tc.tool.Description() || string(spec.Schema) != string(tc.tool.Schema()) {
			t.Fatalf("%s spec descriptor = %+v, want live tool descriptor", tc.tool.Name(), spec.ToolDescriptor)
		}
		if spec.AuditKind != "memory" || !spec.PromptSafe || spec.Mutating != tc.mutating || spec.Idempotent != tc.idempotent {
			t.Fatalf("%s spec = %+v, want memory spec mutating/idempotent %v/%v", tc.tool.Name(), spec, tc.mutating, tc.idempotent)
		}
	}
}

func containsTag(tags []string, query string) bool {
	for _, tag := range tags {
		if tag == query {
			return true
		}
	}
	return false
}

func containsContent(content string, query string) bool {
	content = strings.ToLower(content)
	query = strings.ToLower(query)
	return strings.Contains(content, query) || strings.Contains(query, content)
}

func executeTool(t *testing.T, ctx context.Context, tool toolmeta.Tool, args string) map[string]any {
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

func entriesField(t *testing.T, out map[string]any, name string) []Entry {
	t.Helper()
	raw, err := json.Marshal(out[name])
	if err != nil {
		t.Fatalf("marshal %s: %v", name, err)
	}
	var entries []Entry
	if err := json.Unmarshal(raw, &entries); err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
	return entries
}

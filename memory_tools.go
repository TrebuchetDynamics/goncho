package goncho

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"


	toolmeta "github.com/TrebuchetDynamics/goncho/toolmeta"
)

// MemoryToolStore abstracts the storage backend for agent-controlled memory
// tool calls.
type MemoryToolStore interface {
	Store(ctx context.Context, entry MemoryToolEntry) error
	Retrieve(ctx context.Context, query string, limit int) ([]MemoryToolEntry, error)
	Update(ctx context.Context, id string, content string) error
	Forget(ctx context.Context, id string) error
}

type MemoryImportanceUpdater interface {
	UpdateImportance(ctx context.Context, id string, importance float64) error
}

// MemoryToolEntry is a single unit of agent-managed memory.
type MemoryToolEntry struct {
	ID         string            `json:"id"`
	Content    string            `json:"content"`
	Tags       []string          `json:"tags"`
	Importance float64           `json:"importance"`
	SessionID  string            `json:"session_id,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// memoryToolBase provides common fields for memory tool implementations.
type memoryToolBase struct {
	store MemoryToolStore
}

func newMemoryToolBase(store MemoryToolStore) memoryToolBase {
	return memoryToolBase{store: store}
}

type storeMemoryTool struct {
	memoryToolBase
}

type StoreMemoryTool struct {
	storeMemoryTool
}

func NewStoreMemoryTool(store MemoryToolStore) *StoreMemoryTool {
	return &StoreMemoryTool{storeMemoryTool{newMemoryToolBase(store)}}
}

func (t *storeMemoryTool) Name() string           { return "store_memory" }
func (t *storeMemoryTool) Timeout() time.Duration { return 5 * time.Second }
func (t *storeMemoryTool) Description() string {
	return "Persist information to agent memory. Use to remember facts, preferences, and lessons that should survive across sessions."
}
func (t *storeMemoryTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"content":{"type":"string","description":"The information to store"},"tags":{"type":"array","items":{"type":"string"},"description":"Tags for categorization"},"importance":{"type":"number","description":"Importance 0.0-1.0"},"metadata":{"type":"object","additionalProperties":{"type":"string"},"description":"Optional metadata to persist with provenance"}},"required":["content"]}`)
}
func (t storeMemoryTool) Spec() toolmeta.OperationSpec {
	return memoryToolOperationSpec(t.Name(), t.Description(), t.Schema())
}
func (t *storeMemoryTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		Content    string            `json:"content"`
		Tags       []string          `json:"tags"`
		Importance *float64          `json:"importance"`
		Metadata   map[string]string `json:"metadata"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("store_memory: %w", err)
	}
	if in.Content == "" {
		return nil, errors.New("store_memory: content is required")
	}
	importance := 0.5
	if in.Importance != nil {
		importance = clampMemoryImportance(*in.Importance)
	}
	entry := MemoryToolEntry{
		ID:         fmt.Sprintf("mem_%d", time.Now().UnixNano()),
		Content:    in.Content,
		Tags:       in.Tags,
		Importance: importance,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Metadata:   in.Metadata,
	}
	if err := t.store.Store(ctx, entry); err != nil {
		return nil, fmt.Errorf("store_memory: %w", err)
	}
	return json.Marshal(map[string]interface{}{
		"success":          true,
		"id":               entry.ID,
		"message":          "Memory stored.",
		"contract_version": memory.GonchoMemoryV1ContractVersion,
		"local_first":      true,
		"markdown_backed":  true,
		"network_required": false,
		"ollama_required":  false,
	})
}

type retrieveMemoryTool struct {
	memoryToolBase
}

type RetrieveMemoryTool struct {
	retrieveMemoryTool
}

func NewRetrieveMemoryTool(store MemoryToolStore) *RetrieveMemoryTool {
	return &RetrieveMemoryTool{retrieveMemoryTool{newMemoryToolBase(store)}}
}

func (t *retrieveMemoryTool) Name() string           { return "retrieve_memory" }
func (t *retrieveMemoryTool) Timeout() time.Duration { return 5 * time.Second }
func (t *retrieveMemoryTool) Description() string {
	return "Retrieve memories relevant to the given query. Returns ranked results ordered by importance and recency."
}
func (t *retrieveMemoryTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","description":"Search query for memory retrieval"},"limit":{"type":"integer","description":"Max results (default 5)"}},"required":["query"]}`)
}
func (t retrieveMemoryTool) Spec() toolmeta.OperationSpec {
	return memoryToolOperationSpec(t.Name(), t.Description(), t.Schema())
}
func (t *retrieveMemoryTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("retrieve_memory: %w", err)
	}
	if in.Query == "" {
		return nil, errors.New("retrieve_memory: query is required")
	}
	if in.Limit <= 0 {
		in.Limit = 5
	}
	entries, err := t.store.Retrieve(ctx, in.Query, in.Limit)
	if err != nil {
		return nil, fmt.Errorf("retrieve_memory: %w", err)
	}
	if entries == nil {
		entries = []MemoryToolEntry{}
	}
	return json.Marshal(map[string]interface{}{
		"results":          entries,
		"count":            len(entries),
		"contract_version": memory.GonchoMemoryV1ContractVersion,
		"local_first":      true,
		"markdown_backed":  true,
		"network_required": false,
		"ollama_required":  false,
	})
}

type updateMemoryTool struct {
	memoryToolBase
}

type UpdateMemoryTool struct {
	updateMemoryTool
}

func NewUpdateMemoryTool(store MemoryToolStore) *UpdateMemoryTool {
	return &UpdateMemoryTool{updateMemoryTool{newMemoryToolBase(store)}}
}

func (t *updateMemoryTool) Name() string           { return "update_memory" }
func (t *updateMemoryTool) Timeout() time.Duration { return 5 * time.Second }
func (t *updateMemoryTool) Description() string {
	return "Update an existing memory entry. Use when information has changed, needs correction, or its importance should be promoted or demoted."
}
func (t *updateMemoryTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"id":{"type":"string","description":"Memory entry ID to update"},"content":{"type":"string","description":"New content for the memory entry"},"importance":{"type":"number","description":"New importance from 0.0 to 1.0"}},"required":["id"]}`)
}
func (t updateMemoryTool) Spec() toolmeta.OperationSpec {
	return memoryToolOperationSpec(t.Name(), t.Description(), t.Schema())
}
func (t *updateMemoryTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		ID         string   `json:"id"`
		Content    string   `json:"content"`
		Importance *float64 `json:"importance"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("update_memory: %w", err)
	}
	if in.ID == "" {
		return nil, errors.New("update_memory: id is required")
	}
	if in.Content == "" && in.Importance == nil {
		return nil, errors.New("update_memory: content or importance is required")
	}
	if in.Content != "" {
		if err := t.store.Update(ctx, in.ID, in.Content); err != nil {
			return nil, fmt.Errorf("update_memory: %w", err)
		}
	}
	if in.Importance != nil {
		updater, ok := t.store.(MemoryImportanceUpdater)
		if !ok {
			return nil, errors.New("update_memory: store does not support importance updates")
		}
		if err := updater.UpdateImportance(ctx, in.ID, clampMemoryImportance(*in.Importance)); err != nil {
			return nil, fmt.Errorf("update_memory: %w", err)
		}
	}
	return json.Marshal(map[string]interface{}{
		"success":          true,
		"message":          "Memory updated.",
		"contract_version": memory.GonchoMemoryV1ContractVersion,
		"local_first":      true,
		"markdown_backed":  true,
		"network_required": false,
		"ollama_required":  false,
	})
}

type summarizeMemoryTool struct {
	memoryToolBase
}

type SummarizeMemoryTool struct {
	summarizeMemoryTool
}

func NewSummarizeMemoryTool(store MemoryToolStore) *SummarizeMemoryTool {
	return &SummarizeMemoryTool{summarizeMemoryTool{newMemoryToolBase(store)}}
}

func (t *summarizeMemoryTool) Name() string           { return "summarize_memories" }
func (t *summarizeMemoryTool) Timeout() time.Duration { return 10 * time.Second }
func (t *summarizeMemoryTool) Description() string {
	return "Summarize related memories by tag or query. Compresses multiple entries into a consolidated summary."
}
func (t *summarizeMemoryTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"filter":{"type":"string","description":"Tag or query to filter memories for summarization"},"max_items":{"type":"integer","description":"Max entries to summarize (default 10)"}},"required":["filter"]}`)
}
func (t summarizeMemoryTool) Spec() toolmeta.OperationSpec {
	return memoryToolOperationSpec(t.Name(), t.Description(), t.Schema())
}
func (t *summarizeMemoryTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		Filter   string `json:"filter"`
		MaxItems int    `json:"max_items"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("summarize_memories: %w", err)
	}
	if in.Filter == "" {
		return nil, errors.New("summarize_memories: filter is required")
	}
	if in.MaxItems <= 0 {
		in.MaxItems = 10
	}
	entries, err := t.store.Retrieve(ctx, in.Filter, in.MaxItems)
	if err != nil {
		return nil, fmt.Errorf("summarize_memories: %w", err)
	}
	if entries == nil {
		entries = []MemoryToolEntry{}
	}
	return json.Marshal(map[string]interface{}{
		"summarized":       len(entries),
		"filter":           in.Filter,
		"summary":          summarizeMemoryEntries(entries),
		"message":          "Memories retrieved for summarization.",
		"contract_version": memory.GonchoMemoryV1ContractVersion,
		"local_first":      true,
		"markdown_backed":  true,
		"network_required": false,
		"ollama_required":  false,
	})
}

func summarizeMemoryEntries(entries []MemoryToolEntry) string {
	if len(entries) == 0 {
		return "No matching memories."
	}
	var summary strings.Builder
	for _, entry := range entries {
		content := strings.TrimSpace(entry.Content)
		if content == "" {
			continue
		}
		if summary.Len() > 0 {
			summary.WriteByte('\n')
		}
		summary.WriteString("- ")
		if entry.ID != "" {
			summary.WriteString(entry.ID)
			summary.WriteString(": ")
		}
		summary.WriteString(content)
	}
	if summary.Len() == 0 {
		return "No matching memories."
	}
	return summary.String()
}

type forgetMemoryTool struct {
	memoryToolBase
}

type ForgetMemoryTool struct {
	forgetMemoryTool
}

func NewForgetMemoryTool(store MemoryToolStore) *ForgetMemoryTool {
	return &ForgetMemoryTool{forgetMemoryTool{newMemoryToolBase(store)}}
}

func (t *forgetMemoryTool) Name() string           { return "forget_memory" }
func (t *forgetMemoryTool) Timeout() time.Duration { return 5 * time.Second }
func (t *forgetMemoryTool) Description() string {
	return "Remove a memory entry from active storage (soft delete). Use when information is outdated or no longer relevant."
}
func (t *forgetMemoryTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"id":{"type":"string","description":"Memory entry ID to forget"}},"required":["id"]}`)
}
func (t forgetMemoryTool) Spec() toolmeta.OperationSpec {
	return memoryToolOperationSpec(t.Name(), t.Description(), t.Schema())
}
func (t *forgetMemoryTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	var in struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("forget_memory: %w", err)
	}
	if in.ID == "" {
		return nil, errors.New("forget_memory: id is required")
	}
	if err := t.store.Forget(ctx, in.ID); err != nil {
		return nil, fmt.Errorf("forget_memory: %w", err)
	}
	return json.Marshal(map[string]interface{}{
		"success":          true,
		"message":          "Memory forgotten (soft delete).",
		"contract_version": memory.GonchoMemoryV1ContractVersion,
		"local_first":      true,
		"markdown_backed":  true,
		"network_required": false,
		"ollama_required":  false,
	})
}

func memoryToolOperationSpec(name, description string, schema json.RawMessage) toolmeta.OperationSpec {
	spec, ok := toolmeta.MemoryToolOperationSpec(name)
	if !ok {
		return toolmeta.DefaultSpec(name, description, schema)
	}
	spec.ToolDescriptor = toolmeta.ToolDescriptor{
		Name:        name,
		Description: description,
		Schema:      schema,
	}
	return spec
}

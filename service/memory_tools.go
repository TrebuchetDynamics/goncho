package goncho

import (
	"context"
	"encoding/json"
	"time"

	"github.com/TrebuchetDynamics/goncho/internal/memorytools"
	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
	"github.com/TrebuchetDynamics/goncho/toolmeta"
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

type memoryToolFacade struct {
	inner interface {
		toolmeta.Tool
		toolmeta.Spec
	}
}

func (t memoryToolFacade) Name() string { return t.inner.Name() }

func (t memoryToolFacade) Description() string { return t.inner.Description() }

func (t memoryToolFacade) Schema() json.RawMessage { return t.inner.Schema() }

func (t memoryToolFacade) Timeout() time.Duration { return t.inner.Timeout() }

func (t memoryToolFacade) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	return t.inner.Execute(ctx, args)
}

func (t memoryToolFacade) Spec() toolmeta.OperationSpec { return t.inner.Spec() }

type StoreMemoryTool struct{ memoryToolFacade }

func NewStoreMemoryTool(store MemoryToolStore) *StoreMemoryTool {
	return &StoreMemoryTool{memoryToolFacade{inner: memorytools.NewStoreTool(adaptMemoryToolStore(store))}}
}

type RetrieveMemoryTool struct{ memoryToolFacade }

func NewRetrieveMemoryTool(store MemoryToolStore) *RetrieveMemoryTool {
	return &RetrieveMemoryTool{memoryToolFacade{inner: memorytools.NewRetrieveTool(adaptMemoryToolStore(store))}}
}

type UpdateMemoryTool struct{ memoryToolFacade }

func NewUpdateMemoryTool(store MemoryToolStore) *UpdateMemoryTool {
	return &UpdateMemoryTool{memoryToolFacade{inner: memorytools.NewUpdateTool(adaptMemoryToolStore(store))}}
}

type SummarizeMemoryTool struct{ memoryToolFacade }

func NewSummarizeMemoryTool(store MemoryToolStore) *SummarizeMemoryTool {
	return &SummarizeMemoryTool{memoryToolFacade{inner: memorytools.NewSummarizeTool(adaptMemoryToolStore(store))}}
}

type ForgetMemoryTool struct{ memoryToolFacade }

func NewForgetMemoryTool(store MemoryToolStore) *ForgetMemoryTool {
	return &ForgetMemoryTool{memoryToolFacade{inner: memorytools.NewForgetTool(adaptMemoryToolStore(store))}}
}

type memoryToolStoreAdapter struct {
	store MemoryToolStore
}

type memoryToolImportanceStoreAdapter struct {
	memoryToolStoreAdapter
}

func adaptMemoryToolStore(store MemoryToolStore) memorytools.Store {
	base := memoryToolStoreAdapter{store: store}
	if _, ok := store.(MemoryImportanceUpdater); ok {
		return memoryToolImportanceStoreAdapter{memoryToolStoreAdapter: base}
	}
	return base
}

func (a memoryToolStoreAdapter) Store(ctx context.Context, entry memorytools.Entry) error {
	return a.store.Store(ctx, fromMemoryToolsEntry(entry))
}

func (a memoryToolStoreAdapter) Retrieve(ctx context.Context, query string, limit int) ([]memorytools.Entry, error) {
	entries, err := a.store.Retrieve(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	return sliceutil.Map(entries, toMemoryToolsEntry), nil
}

func (a memoryToolStoreAdapter) Update(ctx context.Context, id string, content string) error {
	return a.store.Update(ctx, id, content)
}

func (a memoryToolStoreAdapter) Forget(ctx context.Context, id string) error {
	return a.store.Forget(ctx, id)
}

func (a memoryToolImportanceStoreAdapter) UpdateImportance(ctx context.Context, id string, importance float64) error {
	return a.store.(MemoryImportanceUpdater).UpdateImportance(ctx, id, importance)
}

func toMemoryToolsEntry(entry MemoryToolEntry) memorytools.Entry {
	return memorytools.Entry{
		ID:         entry.ID,
		Content:    entry.Content,
		Tags:       cloneStrings(entry.Tags),
		Importance: entry.Importance,
		SessionID:  entry.SessionID,
		CreatedAt:  entry.CreatedAt,
		UpdatedAt:  entry.UpdatedAt,
		Metadata:   cloneStringMap(entry.Metadata),
	}
}

func fromMemoryToolsEntry(entry memorytools.Entry) MemoryToolEntry {
	return MemoryToolEntry{
		ID:         entry.ID,
		Content:    entry.Content,
		Tags:       cloneStrings(entry.Tags),
		Importance: entry.Importance,
		SessionID:  entry.SessionID,
		CreatedAt:  entry.CreatedAt,
		UpdatedAt:  entry.UpdatedAt,
		Metadata:   cloneStringMap(entry.Metadata),
	}
}

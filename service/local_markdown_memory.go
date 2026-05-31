package goncho

import (
	"context"
	"database/sql"

	"github.com/TrebuchetDynamics/goncho/internal/localmarkdown"
	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
)

// LocalMarkdownMemoryConfig wires the local-first Goncho V1 memory tools to a
// SQLite database and a Markdown export file.
type LocalMarkdownMemoryConfig = localmarkdown.Config

// LocalMarkdownMemoryStatus is the operator-facing status for the local memory
// backend used by Memory V1 MCP tools.
type LocalMarkdownMemoryStatus = localmarkdown.Status

// LocalMarkdownMemoryStore persists tool memories into the Goncho V1 SQLite
// table and mirrors the table to a human-editable Markdown file.
type LocalMarkdownMemoryStore struct {
	inner *localmarkdown.Store
}

func NewLocalMarkdownMemoryStore(db *sql.DB, cfg LocalMarkdownMemoryConfig) *LocalMarkdownMemoryStore {
	return &LocalMarkdownMemoryStore{inner: localmarkdown.NewStore(db, cfg)}
}

func (s *LocalMarkdownMemoryStore) Status(ctx context.Context) (LocalMarkdownMemoryStatus, error) {
	return s.module().Status(ctx)
}

func (s *LocalMarkdownMemoryStore) Store(ctx context.Context, entry MemoryToolEntry) error {
	return s.module().Store(ctx, toLocalMarkdownEntry(entry))
}

func (s *LocalMarkdownMemoryStore) Retrieve(ctx context.Context, query string, limit int) ([]MemoryToolEntry, error) {
	entries, err := s.module().Retrieve(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	return sliceutil.Map(entries, fromLocalMarkdownEntry), nil
}

func (s *LocalMarkdownMemoryStore) Update(ctx context.Context, id string, content string) error {
	return s.module().Update(ctx, id, content)
}

func (s *LocalMarkdownMemoryStore) UpdateImportance(ctx context.Context, id string, importance float64) error {
	return s.module().UpdateImportance(ctx, id, importance)
}

func (s *LocalMarkdownMemoryStore) Forget(ctx context.Context, id string) error {
	return s.module().Forget(ctx, id)
}

func (s *LocalMarkdownMemoryStore) module() *localmarkdown.Store {
	if s == nil {
		return nil
	}
	return s.inner
}

func toLocalMarkdownEntry(entry MemoryToolEntry) localmarkdown.Entry {
	return localmarkdown.Entry{
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

func fromLocalMarkdownEntry(entry localmarkdown.Entry) MemoryToolEntry {
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

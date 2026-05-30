package goncho

import (
	"context"
	"strings"
	"sync"

	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
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
	for _, entry := range m.entries {
		if query == "" || containsTag(entry.Tags, query) || containsMemoryContent(entry.Content, query) {
			results = append(results, entry)
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
	return sliceutil.Contains(tags, query)
}

func containsMemoryContent(content string, query string) bool {
	content = strings.ToLower(content)
	query = strings.ToLower(query)
	return strings.Contains(content, query) || strings.Contains(query, content)
}

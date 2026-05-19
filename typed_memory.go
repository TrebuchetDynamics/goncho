package goncho

import (
	"fmt"
	"time"
)

type MemoryType string

const (
	MemoryTypeIdentity   MemoryType = "identity"
	MemoryTypePreference MemoryType = "preference"
	MemoryTypeGoal       MemoryType = "goal"
	MemoryTypeHabit      MemoryType = "habit"
	MemoryTypeEpisode    MemoryType = "episode"
	MemoryTypeReflection MemoryType = "reflection"
)

type MemoryEntry struct {
	ID         string
	Type       MemoryType
	Content    string
	Confidence float64
	Durability float64
	CreatedAt  time.Time
	UpdatedAt  time.Time
	LastUsedAt time.Time
}

type TypedMemoryStore struct {
	entries map[string]*MemoryEntry
}

func NewTypedMemoryStore() *TypedMemoryStore {
	return &TypedMemoryStore{
		entries: make(map[string]*MemoryEntry),
	}
}

func (s *TypedMemoryStore) Create(entry *MemoryEntry) error {
	if entry.ID == "" {
		entry.ID = fmt.Sprintf("%s-%d", entry.Type, time.Now().UnixNano())
	}
	if entry.Confidence < 0 || entry.Confidence > 1 {
		return fmt.Errorf("confidence must be between 0 and 1, got %f", entry.Confidence)
	}
	if entry.Durability < 0 || entry.Durability > 1 {
		return fmt.Errorf("durability must be between 0 and 1, got %f", entry.Durability)
	}
	now := time.Now()
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = now
	}
	entry.UpdatedAt = now
	if entry.LastUsedAt.IsZero() {
		entry.LastUsedAt = now
	}
	s.entries[entry.ID] = entry
	return nil
}

func (s *TypedMemoryStore) Read(id string) (*MemoryEntry, bool) {
	entry, ok := s.entries[id]
	if !ok {
		return nil, false
	}
	entry.LastUsedAt = time.Now()
	return entry, true
}

func (s *TypedMemoryStore) Update(id string, updateFn func(*MemoryEntry)) bool {
	entry, ok := s.entries[id]
	if !ok {
		return false
	}
	updateFn(entry)
	entry.UpdatedAt = time.Now()
	entry.LastUsedAt = time.Now()
	return true
}

func (s *TypedMemoryStore) Delete(id string) bool {
	_, ok := s.entries[id]
	if !ok {
		return false
	}
	delete(s.entries, id)
	return true
}

func (s *TypedMemoryStore) ListByType(memType MemoryType) []*MemoryEntry {
	var result []*MemoryEntry
	for _, entry := range s.entries {
		if entry.Type == memType {
			result = append(result, entry)
		}
	}
	return result
}

func (s *TypedMemoryStore) ListAll() []*MemoryEntry {
	var result []*MemoryEntry
	for _, entry := range s.entries {
		result = append(result, entry)
	}
	return result
}

func (s *TypedMemoryStore) Prune(now time.Time) int {
	pruned := 0
	for id, entry := range s.entries {
		if shouldPrune(entry, now) {
			delete(s.entries, id)
			pruned++
		}
	}
	return pruned
}

func shouldPrune(entry *MemoryEntry, now time.Time) bool {
	activeStale := 21 * 24 * time.Hour
	durableDecay := 120 * 24 * time.Hour

	if entry.Durability < 0.5 {
		return now.Sub(entry.LastUsedAt) > activeStale
	}
	return now.Sub(entry.LastUsedAt) > durableDecay
}

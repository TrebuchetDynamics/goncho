package typedmemory

import (
	"testing"
	"time"
)

func TestTypedMemory_CRUD(t *testing.T) {
	store := NewTypedMemoryStore()

	entry := &MemoryEntry{
		Type:       MemoryTypeIdentity,
		Content:    "User is a software engineer",
		Confidence: 0.9,
		Durability: 0.8,
	}

	if err := store.Create(entry); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if entry.ID == "" {
		t.Fatal("expected ID to be generated")
	}

	read, ok := store.Read(entry.ID)
	if !ok {
		t.Fatal("expected to find entry")
	}
	if read.Content != entry.Content {
		t.Fatalf("content mismatch: %s vs %s", read.Content, entry.Content)
	}

	updated := store.Update(entry.ID, func(e *MemoryEntry) {
		e.Confidence = 0.95
	})
	if !updated {
		t.Fatal("expected update to succeed")
	}

	read, _ = store.Read(entry.ID)
	if read.Confidence != 0.95 {
		t.Fatalf("expected confidence=0.95, got %f", read.Confidence)
	}

	deleted := store.Delete(entry.ID)
	if !deleted {
		t.Fatal("expected delete to succeed")
	}

	_, ok = store.Read(entry.ID)
	if ok {
		t.Fatal("expected entry to be deleted")
	}
}

func TestTypedMemory_ListByType(t *testing.T) {
	store := NewTypedMemoryStore()
	store.Create(&MemoryEntry{Type: MemoryTypeIdentity, Content: "id1", Confidence: 0.5})
	store.Create(&MemoryEntry{Type: MemoryTypePreference, Content: "pref1", Confidence: 0.5})
	store.Create(&MemoryEntry{Type: MemoryTypeIdentity, Content: "id2", Confidence: 0.5})

	identities := store.ListByType(MemoryTypeIdentity)
	if len(identities) != 2 {
		t.Fatalf("expected 2 identities, got %d", len(identities))
	}

	preferences := store.ListByType(MemoryTypePreference)
	if len(preferences) != 1 {
		t.Fatalf("expected 1 preference, got %d", len(preferences))
	}
}

func TestTypedMemory_ConfidenceValidation(t *testing.T) {
	store := NewTypedMemoryStore()
	entry := &MemoryEntry{Type: MemoryTypeIdentity, Content: "test", Confidence: 1.5}
	if err := store.Create(entry); err == nil {
		t.Fatal("expected error for confidence > 1")
	}

	entry.Confidence = -0.1
	if err := store.Create(entry); err == nil {
		t.Fatal("expected error for confidence < 0")
	}
}

func TestTypedMemory_DurabilityValidation(t *testing.T) {
	store := NewTypedMemoryStore()
	entry := &MemoryEntry{Type: MemoryTypeIdentity, Content: "test", Confidence: 0.5, Durability: 1.5}
	if err := store.Create(entry); err == nil {
		t.Fatal("expected error for durability > 1")
	}
}

func TestTypedMemory_AllTypes(t *testing.T) {
	store := NewTypedMemoryStore()
	types := []MemoryType{MemoryTypeIdentity, MemoryTypePreference, MemoryTypeGoal, MemoryTypeHabit, MemoryTypeEpisode, MemoryTypeReflection}

	for _, memType := range types {
		store.Create(&MemoryEntry{Type: memType, Content: string(memType), Confidence: 0.5})
	}

	all := store.ListAll()
	if len(all) != 6 {
		t.Fatalf("expected 6 entries, got %d", len(all))
	}

	stats := store.GetStats()
	if len(stats) != 6 {
		t.Fatalf("expected 6 types in stats, got %d", len(stats))
	}
}

func TestTypedMemory_PruneActiveStale(t *testing.T) {
	store := NewTypedMemoryStore()
	now := time.Now()
	old := now.Add(-30 * 24 * time.Hour)

	store.Create(&MemoryEntry{
		Type:       MemoryTypePreference,
		Content:    "old preference",
		Confidence: 0.5,
		Durability: 0.3,
		LastUsedAt: old,
	})

	pruned := store.Prune(now)
	if pruned != 1 {
		t.Fatalf("expected 1 pruned, got %d", pruned)
	}
}

func TestTypedMemory_PruneDurableDecay(t *testing.T) {
	store := NewTypedMemoryStore()
	now := time.Now()
	old := now.Add(-150 * 24 * time.Hour)

	store.Create(&MemoryEntry{
		Type:       MemoryTypeIdentity,
		Content:    "old identity",
		Confidence: 0.9,
		Durability: 0.9,
		LastUsedAt: old,
	})

	pruned := store.Prune(now)
	if pruned != 1 {
		t.Fatalf("expected 1 pruned, got %d", pruned)
	}
}

func TestTypedMemory_NoPruneRecent(t *testing.T) {
	store := NewTypedMemoryStore()
	now := time.Now()

	store.Create(&MemoryEntry{
		Type:       MemoryTypeHabit,
		Content:    "recent habit",
		Confidence: 0.5,
		Durability: 0.3,
		LastUsedAt: now.Add(-5 * 24 * time.Hour),
	})

	pruned := store.Prune(now)
	if pruned != 0 {
		t.Fatalf("expected 0 pruned, got %d", pruned)
	}
}

func TestTypedMemory_AverageConfidence(t *testing.T) {
	store := NewTypedMemoryStore()
	store.Create(&MemoryEntry{Type: MemoryTypeGoal, Content: "g1", Confidence: 0.8})
	store.Create(&MemoryEntry{Type: MemoryTypeGoal, Content: "g2", Confidence: 0.6})

	avg := store.GetAverageConfidence(MemoryTypeGoal)
	if avg != 0.7 {
		t.Fatalf("expected avg=0.7, got %f", avg)
	}
}

func TestTypedMemory_PruningCandidates(t *testing.T) {
	store := NewTypedMemoryStore()
	now := time.Now()

	store.Create(&MemoryEntry{Type: MemoryTypeEpisode, Content: "old", Confidence: 0.5, Durability: 0.3, LastUsedAt: now.Add(-30 * 24 * time.Hour)})
	store.Create(&MemoryEntry{Type: MemoryTypeEpisode, Content: "recent", Confidence: 0.5, Durability: 0.3, LastUsedAt: now.Add(-5 * 24 * time.Hour)})

	candidates := store.GetPruningCandidates(now)
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(candidates))
	}
	if candidates[0].Content != "old" {
		t.Fatalf("expected old episode, got %s", candidates[0].Content)
	}
}

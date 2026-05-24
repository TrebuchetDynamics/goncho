package typedmemory

import (
	"testing"
	"time"
)

func TestConflictResolution_HigherConfidenceWins(t *testing.T) {
	existing := &MemoryEntry{Type: MemoryTypeIdentity, Content: "old", Confidence: 0.5, UpdatedAt: time.Now()}
	incoming := &MemoryEntry{Type: MemoryTypeIdentity, Content: "new", Confidence: 0.8, UpdatedAt: time.Now()}

	result := ResolveConflict(existing, incoming)
	if !result.Resolved {
		t.Fatal("expected resolved")
	}
	if result.Winner != incoming {
		t.Fatal("expected incoming to win")
	}
	if result.Reason != "higher_confidence" {
		t.Fatalf("expected higher_confidence, got %s", result.Reason)
	}
}

func TestConflictResolution_LowerConfidenceLoses(t *testing.T) {
	existing := &MemoryEntry{Type: MemoryTypeIdentity, Content: "old", Confidence: 0.8, UpdatedAt: time.Now()}
	incoming := &MemoryEntry{Type: MemoryTypeIdentity, Content: "new", Confidence: 0.5, UpdatedAt: time.Now()}

	result := ResolveConflict(existing, incoming)
	if !result.Resolved {
		t.Fatal("expected resolved")
	}
	if result.Winner != existing {
		t.Fatal("expected existing to win")
	}
	if result.Reason != "lower_confidence" {
		t.Fatalf("expected lower_confidence, got %s", result.Reason)
	}
}

func TestConflictResolution_EqualConfidenceNewerWins(t *testing.T) {
	now := time.Now()
	existing := &MemoryEntry{Type: MemoryTypeIdentity, Content: "old", Confidence: 0.7, UpdatedAt: now.Add(-1 * time.Hour)}
	incoming := &MemoryEntry{Type: MemoryTypeIdentity, Content: "new", Confidence: 0.7, UpdatedAt: now}

	result := ResolveConflict(existing, incoming)
	if !result.Resolved {
		t.Fatal("expected resolved")
	}
	if result.Winner != incoming {
		t.Fatal("expected incoming to win")
	}
	if result.Reason != "equal_confidence_newer" {
		t.Fatalf("expected equal_confidence_newer, got %s", result.Reason)
	}
}

func TestConflictResolution_EqualConfidenceOlderLoses(t *testing.T) {
	now := time.Now()
	existing := &MemoryEntry{Type: MemoryTypeIdentity, Content: "old", Confidence: 0.7, UpdatedAt: now}
	incoming := &MemoryEntry{Type: MemoryTypeIdentity, Content: "new", Confidence: 0.7, UpdatedAt: now.Add(-1 * time.Hour)}

	result := ResolveConflict(existing, incoming)
	if !result.Resolved {
		t.Fatal("expected resolved")
	}
	if result.Winner != existing {
		t.Fatal("expected existing to win")
	}
	if result.Reason != "equal_confidence_older" {
		t.Fatalf("expected equal_confidence_older, got %s", result.Reason)
	}
}

func TestConflictResolution_NoExisting(t *testing.T) {
	incoming := &MemoryEntry{Type: MemoryTypeIdentity, Content: "new", Confidence: 0.5}
	result := ResolveConflict(nil, incoming)
	if !result.Resolved {
		t.Fatal("expected resolved")
	}
	if result.Winner != incoming {
		t.Fatal("expected incoming to win")
	}
}

func TestConflictResolution_TypeMismatch(t *testing.T) {
	existing := &MemoryEntry{Type: MemoryTypeIdentity, Content: "id"}
	incoming := &MemoryEntry{Type: MemoryTypePreference, Content: "pref"}
	result := ResolveConflict(existing, incoming)
	if result.Resolved {
		t.Fatal("expected not resolved for type mismatch")
	}
}

func TestTypedMemory_UpsertWithConflictResolution(t *testing.T) {
	store := NewTypedMemoryStore()
	store.Create(&MemoryEntry{Type: MemoryTypeGoal, Content: "old goal", Confidence: 0.5})

	result := store.UpsertWithConflictResolution(&MemoryEntry{Type: MemoryTypeGoal, Content: "old goal", Confidence: 0.8})
	if !result.Resolved {
		t.Fatal("expected resolved")
	}
	if result.Winner.Content != "old goal" {
		t.Fatal("expected winner to have old goal content")
	}
	if result.Winner.Confidence != 0.8 {
		t.Fatalf("expected confidence updated to 0.8, got %f", result.Winner.Confidence)
	}
}

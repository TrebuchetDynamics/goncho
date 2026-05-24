package goncho

import (
	"reflect"
	"testing"
	"time"
)

func TestImportancePublicFacadeRanksAndReviewsMemoryToolEntries(t *testing.T) {
	s := NewImportanceScorer()
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	entries := []MemoryToolEntry{
		{ID: "old-high", Content: "latency plan", Importance: 0.95, CreatedAt: now.Add(-90 * 24 * time.Hour), UpdatedAt: now.Add(-90 * 24 * time.Hour)},
		{ID: "fresh-low", Content: "latency note", Importance: 0.2, CreatedAt: now.Add(-1 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)},
		{ID: "fresh-important", Content: "latency SLO", Importance: 0.8, CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)},
	}

	ranked := s.Rank(entries, map[string]float64{
		"old-high":        0.9,
		"fresh-low":       0.9,
		"fresh-important": 0.9,
	}, now)
	var got []string
	for _, item := range ranked {
		got = append(got, item.Entry.ID)
	}
	want := []string{"fresh-important", "fresh-low", "old-high"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ranked IDs = %v, want %v", got, want)
	}

	candidates := s.ReviewRetentionCandidates([]MemoryToolEntry{
		{ID: "old-low", Content: "low", Importance: 0.1, CreatedAt: now.Add(-120 * 24 * time.Hour), UpdatedAt: now.Add(-120 * 24 * time.Hour)},
		{ID: "fresh-low", Content: "fresh", Importance: 0.1, CreatedAt: now, UpdatedAt: now},
	}, RetentionPolicy{
		Now:                    now,
		MinAge:                 30 * 24 * time.Hour,
		MinEffectiveImportance: 0.05,
	})
	if len(candidates) != 1 || candidates[0].Entry.ID != "old-low" || candidates[0].Action != RetentionActionForget {
		t.Fatalf("retention candidates = %+v, want old-low forget candidate", candidates)
	}
	if DefaultDecayCurve(now.Add(-30*24*time.Hour), now) < 0.45 {
		t.Fatalf("DefaultDecayCurve after one half-life should stay available through root facade")
	}
}

package importance

import (
	"reflect"
	"testing"
	"time"
)

func TestImportanceScorer_Score(t *testing.T) {
	s := NewScorer()
	now := time.Now()

	entry := Entry{
		ID: "mem_1", Content: "test", Importance: 0.8, CreatedAt: now,
	}
	score := s.Score(entry, 0.5, now)
	if score <= 0 || score > 1 {
		t.Fatalf("score = %f, want 0 < score <= 1", score)
	}
}

func TestImportanceScorer_Decay(t *testing.T) {
	s := NewScorer()
	now := time.Now()
	old := now.Add(-60 * 24 * time.Hour) // 60 days ago

	recent := Entry{ID: "r", Content: "recent", Importance: 0.5, CreatedAt: now}
	stale := Entry{ID: "s", Content: "stale", Importance: 0.5, CreatedAt: old}

	recentScore := s.Score(recent, 0.5, now)
	staleScore := s.Score(stale, 0.5, now)

	if staleScore >= recentScore {
		t.Fatalf("stale score %f >= recent score %f, want decay to reduce old entry score", staleScore, recentScore)
	}
}

func TestImportanceScorer_HighImportance(t *testing.T) {
	s := NewScorer()
	now := time.Now()

	low := Entry{ID: "l", Content: "low", Importance: 0.1, CreatedAt: now}
	high := Entry{ID: "h", Content: "high", Importance: 0.9, CreatedAt: now}

	if s.Score(high, 0.5, now) <= s.Score(low, 0.5, now) {
		t.Fatal("high importance entry should score higher than low importance")
	}
}

func TestImportanceScorer_RankOrdersByRelevanceImportanceAndRecency(t *testing.T) {
	s := NewScorer()
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	entries := []Entry{
		{ID: "old-high", Content: "latency plan", Importance: 0.95, CreatedAt: now.Add(-90 * 24 * time.Hour), UpdatedAt: now.Add(-90 * 24 * time.Hour)},
		{ID: "fresh-low", Content: "latency note", Importance: 0.2, CreatedAt: now.Add(-1 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)},
		{ID: "fresh-important", Content: "latency SLO", Importance: 0.8, CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)},
		{ID: "irrelevant", Content: "theme preference", Importance: 0.1, CreatedAt: now.Add(-120 * 24 * time.Hour), UpdatedAt: now.Add(-120 * 24 * time.Hour)},
	}

	ranked := s.Rank(entries, map[string]float64{
		"old-high":        0.9,
		"fresh-low":       0.9,
		"fresh-important": 0.9,
		"irrelevant":      0.0,
	}, now)

	var got []string
	for _, item := range ranked {
		got = append(got, item.Entry.ID)
	}
	want := []string{"fresh-important", "fresh-low", "old-high", "irrelevant"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ranked IDs = %v, want %v with α·recency + β·effective_importance + γ·relevance", got, want)
	}
	if ranked[2].EffectiveImportance >= ranked[1].EffectiveImportance {
		t.Fatalf("old high-importance memory effective importance = %f, fresh low = %f; want decay to reduce old unused memory", ranked[2].EffectiveImportance, ranked[1].EffectiveImportance)
	}
}

func TestDefaultDecayCurve_HalfLife(t *testing.T) {
	now := time.Now()
	oneHalfLife := now.Add(-30 * 24 * time.Hour)

	val := DefaultDecayCurve(oneHalfLife, now)
	if val < 0.45 || val > 0.55 {
		t.Fatalf("after one half-life, decay = %f, want ~0.5", val)
	}
}

func TestDecayCurve_ReviewCandidatesFindsLowImportanceOldMemories(t *testing.T) {
	s := NewScorer()
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	entries := []Entry{
		{ID: "fresh-low", Content: "fresh", Importance: 0.1, CreatedAt: now, UpdatedAt: now},
		{ID: "old-important", Content: "important", Importance: 0.95, CreatedAt: now.Add(-120 * 24 * time.Hour), UpdatedAt: now.Add(-120 * 24 * time.Hour)},
		{ID: "old-low", Content: "low", Importance: 0.1, CreatedAt: now.Add(-120 * 24 * time.Hour), UpdatedAt: now.Add(-120 * 24 * time.Hour)},
	}

	candidates := s.ReviewRetentionCandidates(entries, RetentionPolicy{
		Now:                    now,
		MinAge:                 30 * 24 * time.Hour,
		MinEffectiveImportance: 0.05,
	})
	if len(candidates) != 1 {
		t.Fatalf("candidates = %+v, want one old low-importance candidate", candidates)
	}
	if candidates[0].Entry.ID != "old-low" || candidates[0].Action != RetentionActionForget {
		t.Fatalf("candidate = %+v, want old-low forget candidate", candidates[0])
	}
	if candidates[0].EffectiveImportance > 0.05 {
		t.Fatalf("candidate effective importance = %f, want below threshold", candidates[0].EffectiveImportance)
	}
}

func TestEffectiveImportance(t *testing.T) {
	s := NewScorer()
	now := time.Now()

	entry := Entry{ID: "e", Content: "test", Importance: 0.8, CreatedAt: now}
	eff := s.EffectiveImportance(entry, now)

	if eff < 0.79 {
		t.Fatalf("effective importance = %f for fresh entry with 0.8 importance, want ~0.8", eff)
	}
}

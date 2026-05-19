package goncho

import (
	"math"
	"sort"
	"strings"
	"time"
)

const defaultDecayHalfLife = 30 * 24 * time.Hour // 30 days

type ImportanceScorer struct {
	alphaRecency   float64
	betaImportance float64
	gammaRelevance float64
	decayHalfLife  time.Duration
}

type ScoredMemory struct {
	Entry               MemoryToolEntry
	Relevance           float64
	Recency             float64
	EffectiveImportance float64
	Score               float64
}

func NewImportanceScorer() *ImportanceScorer {
	return &ImportanceScorer{
		alphaRecency:   0.3,
		betaImportance: 0.5,
		gammaRelevance: 0.2,
		decayHalfLife:  defaultDecayHalfLife,
	}
}

func (s *ImportanceScorer) Score(entry MemoryToolEntry, relevanceScore float64, now time.Time) float64 {
	recency := s.recencyScore(memoryReferenceTime(entry), now)
	effectiveImportance := s.EffectiveImportance(entry, now)
	score := s.alphaRecency*recency + s.betaImportance*effectiveImportance + s.gammaRelevance*clamp01(relevanceScore)
	return clamp01(score)
}

func (s *ImportanceScorer) Rank(entries []MemoryToolEntry, relevanceByID map[string]float64, now time.Time) []ScoredMemory {
	if s == nil {
		s = NewImportanceScorer()
	}
	out := make([]ScoredMemory, 0, len(entries))
	for _, entry := range entries {
		relevance := relevanceByID[entry.ID]
		out = append(out, ScoredMemory{
			Entry:               entry,
			Relevance:           clamp01(relevance),
			Recency:             s.recencyScore(memoryReferenceTime(entry), now),
			EffectiveImportance: s.EffectiveImportance(entry, now),
			Score:               s.Score(entry, relevance, now),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		if out[i].EffectiveImportance != out[j].EffectiveImportance {
			return out[i].EffectiveImportance > out[j].EffectiveImportance
		}
		iTime := memoryReferenceTime(out[i].Entry)
		jTime := memoryReferenceTime(out[j].Entry)
		if !iTime.Equal(jTime) {
			return iTime.After(jTime)
		}
		return out[i].Entry.ID < out[j].Entry.ID
	})
	return out
}

func (s *ImportanceScorer) RankByQuery(entries []MemoryToolEntry, query string, now time.Time) []ScoredMemory {
	relevance := make(map[string]float64, len(entries))
	for _, entry := range entries {
		relevance[entry.ID] = MemoryEntryRelevance(entry, query)
	}
	return s.Rank(entries, relevance, now)
}

func (s *ImportanceScorer) recencyScore(createdAt time.Time, now time.Time) float64 {
	age := now.Sub(createdAt)
	if age <= 0 {
		return 1.0
	}
	halfLives := float64(age) / float64(s.decayHalfLife)
	return math.Exp2(-halfLives)
}

func (s *ImportanceScorer) EffectiveImportance(entry MemoryToolEntry, now time.Time) float64 {
	base := clamp01(entry.Importance) * s.recencyScore(memoryReferenceTime(entry), now)
	if base < 0.01 {
		base = 0.01
	}
	return base
}

func DefaultDecayCurve(createdAt time.Time, now time.Time) float64 {
	age := now.Sub(createdAt)
	if age <= 0 {
		return 1.0
	}
	halfLives := float64(age) / float64(defaultDecayHalfLife)
	return math.Exp2(-halfLives)
}

func MemoryEntryRelevance(entry MemoryToolEntry, query string) float64 {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return 0
	}
	content := strings.ToLower(entry.Content)
	if strings.Contains(content, query) {
		return 1
	}
	tokens := strings.Fields(query)
	if len(tokens) == 0 {
		return 0
	}
	hits := 0
	for _, token := range tokens {
		if strings.Contains(content, token) {
			hits++
			continue
		}
		for _, tag := range entry.Tags {
			if strings.Contains(strings.ToLower(tag), token) {
				hits++
				break
			}
		}
	}
	return clamp01(float64(hits) / float64(len(tokens)))
}

func memoryReferenceTime(entry MemoryToolEntry) time.Time {
	if !entry.UpdatedAt.IsZero() {
		return entry.UpdatedAt
	}
	if !entry.CreatedAt.IsZero() {
		return entry.CreatedAt
	}
	return time.Now().UTC()
}

func clamp01(value float64) float64 {
	switch {
	case value < 0:
		return 0
	case value > 1:
		return 1
	default:
		return value
	}
}

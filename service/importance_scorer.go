package goncho

import (
	"time"

	"github.com/TrebuchetDynamics/goncho/internal/importance"
	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
)

const defaultDecayHalfLife = importance.DefaultDecayHalfLife

type ImportanceScorer struct {
	inner *importance.Scorer
}

type ScoredMemory struct {
	Entry               MemoryToolEntry
	Relevance           float64
	Recency             float64
	EffectiveImportance float64
	Score               float64
}

func NewImportanceScorer() *ImportanceScorer {
	return &ImportanceScorer{inner: importance.NewScorer()}
}

func (s *ImportanceScorer) Score(entry MemoryToolEntry, relevanceScore float64, now time.Time) float64 {
	return s.module().Score(toImportanceEntry(entry), relevanceScore, now)
}

func (s *ImportanceScorer) Rank(entries []MemoryToolEntry, relevanceByID map[string]float64, now time.Time) []ScoredMemory {
	ranked := s.module().Rank(toImportanceEntries(entries), relevanceByID, now)
	return sliceutil.Map(ranked, fromScoredImportanceEntry)
}

func (s *ImportanceScorer) RankByQuery(entries []MemoryToolEntry, query string, now time.Time) []ScoredMemory {
	ranked := s.module().RankByQuery(toImportanceEntries(entries), query, now)
	return sliceutil.Map(ranked, fromScoredImportanceEntry)
}

func (s *ImportanceScorer) EffectiveImportance(entry MemoryToolEntry, now time.Time) float64 {
	return s.module().EffectiveImportance(toImportanceEntry(entry), now)
}

func DefaultDecayCurve(createdAt time.Time, now time.Time) float64 {
	return importance.DefaultDecayCurve(createdAt, now)
}

func MemoryEntryRelevance(entry MemoryToolEntry, query string) float64 {
	return importance.MemoryEntryRelevance(toImportanceEntry(entry), query)
}

func (s *ImportanceScorer) module() *importance.Scorer {
	if s == nil || s.inner == nil {
		return importance.NewScorer()
	}
	return s.inner
}

func toImportanceEntries(entries []MemoryToolEntry) []importance.Entry {
	return sliceutil.Map(entries, toImportanceEntry)
}

func toImportanceEntry(entry MemoryToolEntry) importance.Entry {
	return importance.Entry{
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

func fromImportanceEntry(entry importance.Entry) MemoryToolEntry {
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

func fromScoredImportanceEntry(item importance.ScoredEntry) ScoredMemory {
	return ScoredMemory{
		Entry:               fromImportanceEntry(item.Entry),
		Relevance:           item.Relevance,
		Recency:             item.Recency,
		EffectiveImportance: item.EffectiveImportance,
		Score:               item.Score,
	}
}

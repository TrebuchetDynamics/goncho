package importance

import (
	"math"
	"sort"
	"strings"
	"time"
)

// Entry is the scoring input shape for agent-managed memory.
type Entry struct {
	ID         string            `json:"id"`
	Content    string            `json:"content"`
	Tags       []string          `json:"tags"`
	Importance float64           `json:"importance"`
	SessionID  string            `json:"session_id,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

const DefaultDecayHalfLife = 30 * 24 * time.Hour // 30 days

type Scorer struct {
	alphaRecency   float64
	betaImportance float64
	gammaRelevance float64
	decayHalfLife  time.Duration
}

type ScoredEntry struct {
	Entry               Entry
	Relevance           float64
	Recency             float64
	EffectiveImportance float64
	Score               float64
}

func NewScorer() *Scorer {
	return &Scorer{
		alphaRecency:   0.3,
		betaImportance: 0.5,
		gammaRelevance: 0.2,
		decayHalfLife:  DefaultDecayHalfLife,
	}
}

func (s *Scorer) Score(entry Entry, relevanceScore float64, now time.Time) float64 {
	recency := s.recencyScore(memoryReferenceTime(entry), now)
	effectiveImportance := s.EffectiveImportance(entry, now)
	score := s.alphaRecency*recency + s.betaImportance*effectiveImportance + s.gammaRelevance*clamp01(relevanceScore)
	return clamp01(score)
}

func (s *Scorer) Rank(entries []Entry, relevanceByID map[string]float64, now time.Time) []ScoredEntry {
	if s == nil {
		s = NewScorer()
	}
	out := make([]ScoredEntry, 0, len(entries))
	for _, entry := range entries {
		relevance := relevanceByID[entry.ID]
		out = append(out, ScoredEntry{
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

func (s *Scorer) RankByQuery(entries []Entry, query string, now time.Time) []ScoredEntry {
	relevance := make(map[string]float64, len(entries))
	for _, entry := range entries {
		relevance[entry.ID] = MemoryEntryRelevance(entry, query)
	}
	return s.Rank(entries, relevance, now)
}

func (s *Scorer) recencyScore(createdAt time.Time, now time.Time) float64 {
	age := now.Sub(createdAt)
	if age <= 0 {
		return 1.0
	}
	halfLives := float64(age) / float64(s.decayHalfLife)
	return math.Exp2(-halfLives)
}

func (s *Scorer) EffectiveImportance(entry Entry, now time.Time) float64 {
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
	halfLives := float64(age) / float64(DefaultDecayHalfLife)
	return math.Exp2(-halfLives)
}

func MemoryEntryRelevance(entry Entry, query string) float64 {
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

func memoryReferenceTime(entry Entry) time.Time {
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

type RetentionAction string

const (
	RetentionActionSummarize RetentionAction = "summarize"
	RetentionActionForget    RetentionAction = "forget"
)

type RetentionPolicy struct {
	Now                    time.Time
	MinAge                 time.Duration
	MinEffectiveImportance float64
	Limit                  int
}

type RetentionCandidate struct {
	Entry               Entry
	Age                 time.Duration
	EffectiveImportance float64
	Action              RetentionAction
	Reason              string
}

func (s *Scorer) ReviewRetentionCandidates(entries []Entry, policy RetentionPolicy) []RetentionCandidate {
	if s == nil {
		s = NewScorer()
	}
	now := policy.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	minAge := policy.MinAge
	if minAge <= 0 {
		minAge = DefaultDecayHalfLife
	}
	minEffective := policy.MinEffectiveImportance
	if minEffective <= 0 {
		minEffective = 0.05
	}

	var out []RetentionCandidate
	for _, entry := range entries {
		reference := memoryReferenceTime(entry)
		age := now.Sub(reference)
		if age < minAge {
			continue
		}
		effective := s.EffectiveImportance(entry, now)
		if effective > minEffective {
			continue
		}
		action := RetentionActionForget
		reason := "effective importance below retention threshold"
		if clamp01(entry.Importance) >= 0.3 {
			action = RetentionActionSummarize
			reason = "old memory with decayed importance should be summarized before forgetting"
		}
		out = append(out, RetentionCandidate{
			Entry:               entry,
			Age:                 age,
			EffectiveImportance: effective,
			Action:              action,
			Reason:              reason,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].EffectiveImportance != out[j].EffectiveImportance {
			return out[i].EffectiveImportance < out[j].EffectiveImportance
		}
		if out[i].Age != out[j].Age {
			return out[i].Age > out[j].Age
		}
		return out[i].Entry.ID < out[j].Entry.ID
	})
	if policy.Limit > 0 && len(out) > policy.Limit {
		out = out[:policy.Limit]
	}
	return out
}

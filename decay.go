package goncho

import (
	"sort"
	"time"
)

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
	Entry               MemoryToolEntry
	Age                 time.Duration
	EffectiveImportance float64
	Action              RetentionAction
	Reason              string
}

func (s *ImportanceScorer) ReviewRetentionCandidates(entries []MemoryToolEntry, policy RetentionPolicy) []RetentionCandidate {
	if s == nil {
		s = NewImportanceScorer()
	}
	now := policy.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	minAge := policy.MinAge
	if minAge <= 0 {
		minAge = defaultDecayHalfLife
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

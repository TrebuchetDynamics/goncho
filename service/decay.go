package goncho

import (
	"time"

	"github.com/TrebuchetDynamics/goncho/internal/importance"
)

type RetentionAction = importance.RetentionAction

const (
	RetentionActionSummarize RetentionAction = importance.RetentionActionSummarize
	RetentionActionForget    RetentionAction = importance.RetentionActionForget
)

type RetentionPolicy = importance.RetentionPolicy

type RetentionCandidate struct {
	Entry               MemoryToolEntry
	Age                 time.Duration
	EffectiveImportance float64
	Action              RetentionAction
	Reason              string
}

func (s *ImportanceScorer) ReviewRetentionCandidates(entries []MemoryToolEntry, policy RetentionPolicy) []RetentionCandidate {
	candidates := s.module().ReviewRetentionCandidates(toImportanceEntries(entries), policy)
	out := make([]RetentionCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		out = append(out, RetentionCandidate{
			Entry:               fromImportanceEntry(candidate.Entry),
			Age:                 candidate.Age,
			EffectiveImportance: candidate.EffectiveImportance,
			Action:              candidate.Action,
			Reason:              candidate.Reason,
		})
	}
	return out
}

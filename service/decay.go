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

type RetentionCandidate struct {
	Entry               MemoryToolEntry
	Age                 time.Duration
	EffectiveImportance float64
	Action              RetentionAction
	Reason              string
}

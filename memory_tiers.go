package goncho

import (
	"fmt"
	"strings"
)

type MemoryTier string

const (
	TierGlobal    MemoryTier = "global"
	TierProject   MemoryTier = "project"
	TierTask      MemoryTier = "task"
	TierWorkspace MemoryTier = "workspace"
	TierDecision  MemoryTier = "decision"
)

var ValidMemoryTiers = []MemoryTier{
	TierGlobal, TierProject, TierTask, TierWorkspace, TierDecision,
}

func ValidTier(t string) bool {
	switch MemoryTier(strings.ToLower(strings.TrimSpace(t))) {
	case TierGlobal, TierProject, TierTask, TierWorkspace, TierDecision:
		return true
	default:
		return false
	}
}

func NormalizeTier(raw string) MemoryTier {
	t := MemoryTier(strings.ToLower(strings.TrimSpace(raw)))
	if !ValidTier(string(t)) {
		return TierGlobal
	}
	return t
}

func TierHierarchy() []MemoryTier {
	return []MemoryTier{TierGlobal, TierProject, TierTask, TierWorkspace, TierDecision}
}

func TiersReadableBy(agentTier MemoryTier) []MemoryTier {
	switch agentTier {
	case TierGlobal:
		return []MemoryTier{TierGlobal}
	case TierProject:
		return []MemoryTier{TierGlobal, TierProject}
	case TierTask:
		return []MemoryTier{TierGlobal, TierProject, TierTask}
	case TierWorkspace:
		return []MemoryTier{TierGlobal, TierProject, TierTask, TierWorkspace}
	case TierDecision:
		return []MemoryTier{TierGlobal, TierProject, TierTask, TierWorkspace, TierDecision}
	default:
		return nil
	}
}

func TiersWritableBy(isParent bool) []MemoryTier {
	if isParent {
		return TierHierarchy()
	}
	return []MemoryTier{TierWorkspace}
}

func DefaultTierForSource(sourceKind string) MemoryTier {
	switch strings.ToLower(strings.TrimSpace(sourceKind)) {
	case "manual", "import":
		return TierProject
	case "tool", "runtime":
		return TierTask
	case "derived":
		return TierDecision
	case "reviewed_proposal":
		return TierProject
	default:
		return TierGlobal
	}
}

func ValidateTierOrErr(raw string) error {
	if !ValidTier(raw) {
		return fmt.Errorf("invalid memory tier %q: must be one of global, project, task, workspace, decision", raw)
	}
	return nil
}

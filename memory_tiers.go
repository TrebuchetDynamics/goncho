package goncho

import "github.com/TrebuchetDynamics/goncho/internal/memorypolicy"

type MemoryTier = memorypolicy.Tier

const (
	TierGlobal    = memorypolicy.TierGlobal
	TierProject   = memorypolicy.TierProject
	TierTask      = memorypolicy.TierTask
	TierWorkspace = memorypolicy.TierWorkspace
	TierDecision  = memorypolicy.TierDecision
)

var ValidMemoryTiers = memorypolicy.ValidTiers

func ValidTier(t string) bool {
	return memorypolicy.ValidTier(t)
}

func NormalizeTier(raw string) MemoryTier {
	return memorypolicy.NormalizeTier(raw)
}

func TierHierarchy() []MemoryTier {
	return memorypolicy.Hierarchy()
}

func TiersReadableBy(agentTier MemoryTier) []MemoryTier {
	return memorypolicy.ReadableBy(agentTier)
}

func TiersWritableBy(isParent bool) []MemoryTier {
	return memorypolicy.WritableBy(isParent)
}

func DefaultTierForSource(sourceKind string) MemoryTier {
	return memorypolicy.DefaultTierForSource(sourceKind)
}

func ValidateTierOrErr(raw string) error {
	return memorypolicy.ValidateTierOrErr(raw)
}

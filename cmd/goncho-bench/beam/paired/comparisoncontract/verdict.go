package comparisoncontract

import "github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/paired/comparisoncontract/verdict"

// CI preserves the comparisoncontract confidence-interval API while sharing the
// verdict.CI value type with lower-level conclusion helpers.
type CI = verdict.CI

const (
	ConclusionCandidateSuperior = verdict.ConclusionCandidateSuperior
	ConclusionBaselineSuperior  = verdict.ConclusionBaselineSuperior
	ConclusionInconclusive      = verdict.ConclusionInconclusive

	ReasonCandidateCIAboveEffectFloor = verdict.ReasonCandidateCIAboveEffectFloor
	ReasonBaselineCIBelowEffectFloor  = verdict.ReasonBaselineCIBelowEffectFloor
	ReasonCIOverlapsEffectFloor       = verdict.ReasonCIOverlapsEffectFloor
	ReasonCandidateDeltaAboveFloor    = verdict.ReasonCandidateDeltaAboveFloor
	ReasonBaselineDeltaBelowFloor     = verdict.ReasonBaselineDeltaBelowFloor
	ReasonDeltaWithinEffectFloor      = verdict.ReasonDeltaWithinEffectFloor
)

// Conclusion preserves the comparisoncontract verdict API while delegating to
// the focused verdict contract.
func Conclusion(ci CI, effectSizeFloor float64) (string, string) {
	return verdict.Conclusion(ci, effectSizeFloor)
}

// PointConclusion preserves the comparisoncontract point-verdict API while
// delegating to the focused verdict contract.
func PointConclusion(delta, effectSizeFloor float64) (string, string) {
	return verdict.PointConclusion(delta, effectSizeFloor)
}

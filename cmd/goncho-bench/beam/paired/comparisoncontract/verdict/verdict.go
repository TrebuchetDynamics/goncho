package verdict

// CI is a confidence interval over the paired score delta.
type CI struct {
	Lower float64 `json:"lower"`
	Upper float64 `json:"upper"`
}

const (
	ConclusionCandidateSuperior = "candidate_superior"
	ConclusionBaselineSuperior  = "baseline_superior"
	ConclusionInconclusive      = "inconclusive"

	ReasonCandidateCIAboveEffectFloor = "candidate_ci_above_effect_floor"
	ReasonBaselineCIBelowEffectFloor  = "baseline_ci_below_negative_effect_floor"
	ReasonCIOverlapsEffectFloor       = "ci_overlaps_effect_floor"
	ReasonCandidateDeltaAboveFloor    = "candidate_delta_above_effect_floor"
	ReasonBaselineDeltaBelowFloor     = "baseline_delta_below_negative_effect_floor"
	ReasonDeltaWithinEffectFloor      = "delta_within_effect_floor"
)

func Conclusion(ci CI, effectSizeFloor float64) (string, string) {
	if ci.Lower > effectSizeFloor {
		return ConclusionCandidateSuperior, ReasonCandidateCIAboveEffectFloor
	}
	if ci.Upper < -effectSizeFloor {
		return ConclusionBaselineSuperior, ReasonBaselineCIBelowEffectFloor
	}
	return ConclusionInconclusive, ReasonCIOverlapsEffectFloor
}

func PointConclusion(delta, effectSizeFloor float64) (string, string) {
	if delta > effectSizeFloor {
		return ConclusionCandidateSuperior, ReasonCandidateDeltaAboveFloor
	}
	if delta < -effectSizeFloor {
		return ConclusionBaselineSuperior, ReasonBaselineDeltaBelowFloor
	}
	return ConclusionInconclusive, ReasonDeltaWithinEffectFloor
}

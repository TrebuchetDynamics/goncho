package shared

import "github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared/outcome"

// PairedOutcome is the JSONL contract used by BEAM oracle runs and paired comparisons.
type PairedOutcome = outcome.PairedOutcome

// PairedOutcomeCorrect reports the shared BEAM paired-outcome correctness contract.
func PairedOutcomeCorrect(score float64) bool {
	return outcome.PairedOutcomeCorrect(score)
}

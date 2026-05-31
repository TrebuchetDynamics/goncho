package comparisoncontract

import "github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/paired/comparisoncontract/policy"

const (
	// DefaultBootstrapSamples is the deterministic paired-comparison bootstrap sample count.
	DefaultBootstrapSamples = policy.DefaultBootstrapSamples

	// DefaultEffectSizeFloor is the minimum absolute score delta for a superiority verdict.
	DefaultEffectSizeFloor = policy.DefaultEffectSizeFloor
)

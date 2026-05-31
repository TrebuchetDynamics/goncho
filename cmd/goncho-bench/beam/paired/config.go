package paired

import "github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared"

type Config struct {
	ComparePath             string
	BaselineConfigID        string
	CandidateConfigID       string
	CompareJSONOut          string
	CompareMarkdownOut      string
	CompareBootstrapSamples int
	CompareEffectSizeFloor  float64
	ResultsIn               string
	ResultsOut              string
	ResultsConfigID         string
}

type servicePairedOutcome = shared.PairedOutcome

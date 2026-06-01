package scoring

import goncho "github.com/TrebuchetDynamics/goncho/service"

// CaseScoreFunc supplies the artifact score used by builders that can be
// pure-recall scored or external-judgment scored.
type CaseScoreFunc func(goncho.RecallBenchmarkCaseReport) float64

package oracle

import (
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/judgmentcontract"
	goncho "github.com/TrebuchetDynamics/goncho/service"
)

type beamServiceJudgment = judgmentcontract.Judgment

type beamServiceJudgmentSet = judgmentcontract.Set

type beamServiceJudgmentDiagnostics = judgmentcontract.Diagnostics

func loadBeamServiceJudgments(path string) (*beamServiceJudgmentSet, error) {
	return judgmentcontract.Load(path)
}

func requireCompleteBeamServiceJudgments(judgments beamServiceJudgmentSet, report goncho.RecallBenchmarkReport) error {
	return judgmentcontract.RequireComplete(judgments, report)
}

package oracle

import (
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/oracle/leakagecheck"
	goncho "github.com/TrebuchetDynamics/goncho/service"
)

type beamServiceLeakageChecks = leakagecheck.Checks

func checkBeamServiceLeakage(cases []goncho.RecallBenchmarkServiceCase) beamServiceLeakageChecks {
	return leakagecheck.Check(cases)
}

func beamServiceLeakageContainsText(content, needle string) bool {
	return leakagecheck.ContainsText(content, needle)
}

func beamServiceLeakageContainsID(content, id string) bool {
	return leakagecheck.ContainsID(content, id)
}

func beamServiceHasBlockingLeakage(checks beamServiceLeakageChecks) bool {
	return leakagecheck.HasBlocking(checks)
}

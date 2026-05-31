package operations_test

import (
	"testing"

	"github.com/TrebuchetDynamics/goncho/docs/guards/operations/opguard"
)

func TestServerModeThreatModelDocumentsRequiredControls(t *testing.T) {
	opguard.VerifyFileContract(t, opguard.FileContract{
		Path:  "docs/server-mode-threat-model.md",
		Label: "threat model",
		Markers: []string{
			"auth", "profiles", "workspaces", "audit", "backup", "retention",
			"admin operations", "postgresql adapter", "sqlite remains the reference",
			"no p2p mesh", "non-loopback binds require", "fail closed",
		},
		Fold: true,
	})
}

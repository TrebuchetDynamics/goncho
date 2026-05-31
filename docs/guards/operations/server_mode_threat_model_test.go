package operations_test

import (
	"testing"

	"github.com/TrebuchetDynamics/goncho/docs/guards/guardtest"
)

func TestServerModeThreatModelDocumentsRequiredControls(t *testing.T) {
	doc := guardtest.ReadRepoFile(t, "docs/server-mode-threat-model.md")
	guardtest.ContainsAllFold(t, doc, "threat model", []string{"auth", "profiles", "workspaces", "audit", "backup", "retention", "admin operations", "postgresql adapter", "sqlite remains the reference", "no p2p mesh", "non-loopback binds require", "fail closed"})
}

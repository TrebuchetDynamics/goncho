package docs_test

import "testing"

func TestServerModeThreatModelDocumentsRequiredControls(t *testing.T) {
	doc := mustReadGuardFile(t, "../server-mode-threat-model.md")
	mustContainAllFold(t, doc, "threat model", []string{"auth", "profiles", "workspaces", "audit", "backup", "retention", "admin operations", "postgresql adapter", "sqlite remains the reference", "no p2p mesh", "non-loopback binds require", "fail closed"})
}

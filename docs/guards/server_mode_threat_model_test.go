package docs_test

import (
	"strings"
	"testing"
)

func TestServerModeThreatModelDocumentsRequiredControls(t *testing.T) {
	doc := strings.ToLower(mustReadGuardFile(t, "../server-mode-threat-model.md"))
	for _, want := range []string{"auth", "profiles", "workspaces", "audit", "backup", "retention", "admin operations", "postgresql adapter", "sqlite remains the reference", "no p2p mesh"} {
		if !strings.Contains(doc, want) {
			t.Fatalf("threat model missing %q", want)
		}
	}
	if !strings.Contains(doc, "non-loopback binds require") || !strings.Contains(doc, "fail closed") {
		t.Fatalf("threat model must require fail-closed authenticated non-loopback binds")
	}
}

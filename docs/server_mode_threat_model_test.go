package docs_test

import (
	"os"
	"strings"
	"testing"
)

func TestServerModeThreatModelDocumentsRequiredControls(t *testing.T) {
	raw, err := os.ReadFile("server-mode-threat-model.md")
	if err != nil {
		t.Fatalf("read threat model: %v", err)
	}
	doc := strings.ToLower(string(raw))
	for _, want := range []string{"auth", "profiles", "workspaces", "audit", "backup", "retention", "admin operations", "postgresql adapter", "sqlite remains the reference", "no p2p mesh"} {
		if !strings.Contains(doc, want) {
			t.Fatalf("threat model missing %q", want)
		}
	}
	if !strings.Contains(doc, "non-loopback binds require") || !strings.Contains(doc, "fail closed") {
		t.Fatalf("threat model must require fail-closed authenticated non-loopback binds")
	}
}

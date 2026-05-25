package docs_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConnectorDocsCoverSupportedAndDeferredIntegrations(t *testing.T) {
	want := map[string]string{
		"gormes.md":             "supported-plan",
		"codex.md":              "supported-plan",
		"pi.md":                 "supported-plan",
		"generic-mcp.md":        "supported-local",
		"filesystem-watcher.md": "supported-plan",
		"hermes.md":             "deferred",
		"cursor.md":             "deferred",
		"claude-code.md":        "deferred",
		"opencode.md":           "deferred",
	}
	for file, status := range want {
		path := filepath.Join("integrations", file)
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		doc := strings.ToLower(string(raw))
		for _, marker := range []string{"status: " + status, "local-first", "preview", "goncho-server"} {
			if !strings.Contains(doc, marker) {
				t.Fatalf("%s missing marker %q", path, marker)
			}
		}
	}
}

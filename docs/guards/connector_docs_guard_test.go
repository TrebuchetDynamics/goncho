package docs_test

import (
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
		path := filepath.Join("..", "integrations", file)
		doc := strings.ToLower(mustReadGuardFile(t, path))
		for _, marker := range []string{"status: " + status, "local-first", "preview", "goncho-server"} {
			if !strings.Contains(doc, marker) {
				t.Fatalf("%s missing marker %q", path, marker)
			}
		}
	}
}

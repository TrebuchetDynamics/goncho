package docs_test

import (
	"path/filepath"
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
		mustContainAllFold(t, mustReadGuardFile(t, path), path, []string{"status: " + status, "local-first", "preview", "goncho-server"})
	}
}

package integrations_test

import (
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/goncho/docs/guards/guardtest"
)

func TestConnectorDocsCoverSupportedAndDeferredIntegrations(t *testing.T) {
	want := map[string]string{
		"first-party/gormes.md":       "supported-plan",
		"agent-hosts/codex.md":        "supported-plan",
		"agent-hosts/pi.md":           "supported-plan",
		"local/generic-mcp.md":         "supported-local",
		"local/filesystem-watcher.md":  "supported-plan",
		"deferred/hermes.md":           "deferred",
		"deferred/cursor.md":           "deferred",
		"deferred/claude-code.md":      "deferred",
		"deferred/opencode.md":         "deferred",
	}
	for file, status := range want {
		path := filepath.Join("docs", "integrations", file)
		guardtest.ContainsAllFold(t, guardtest.ReadRepoFile(t, path), path, []string{"status: " + status, "local-first", "preview", "goncho-server"})
	}
}

package localmarkdown_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/goncho/internal/localmarkdown"
	memory "github.com/TrebuchetDynamics/goncho/memory"
)

func TestLocalMarkdownMemoryStorePersistsExportsAndSurvivesRestart(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "memory.db")
	markdownPath := filepath.Join(t.TempDir(), "GONCHO_MEMORY.md")

	sqlite, err := memory.OpenSqlite(dbPath, 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	local := localmarkdown.NewStore(sqlite.DB(), localmarkdown.Config{
		Path:           markdownPath,
		AgentID:        "agent-a",
		WorkspaceID:    "workspace-private",
		ObserverPeerID: "agent-a",
		PeerID:         "user-juan",
		SessionID:      "telegram:1",
	})

	status, err := local.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !status.Enabled || status.NetworkRequired || status.OllamaRequired {
		t.Fatalf("status = %+v, want enabled local-only markdown memory", status)
	}
	if !stringSliceContains(status.MCPTools, "store_memory") || stringSliceContains(status.MCPTools, "purge_memory") {
		t.Fatalf("MCPTools = %#v, want normal V1 tools without purge", status.MCPTools)
	}

	if err := local.Store(ctx, localmarkdown.Entry{
		ID:         "mem_manual_local_1",
		Content:    "Juan wants Goncho memory to be local markdown and fast.",
		Tags:       []string{"goncho", "local"},
		Importance: 0.9,
	}); err != nil {
		t.Fatalf("Store: %v", err)
	}
	assertFileContains(t, markdownPath, "mem_manual_local_1")
	assertFileContains(t, markdownPath, "local markdown and fast")

	if err := sqlite.Close(ctx); err != nil {
		t.Fatalf("Close: %v", err)
	}
	reopened, err := memory.OpenSqlite(dbPath, 0, nil)
	if err != nil {
		t.Fatalf("reopen sqlite: %v", err)
	}
	defer reopened.Close(ctx)
	restarted := localmarkdown.NewStore(reopened.DB(), localmarkdown.Config{
		Path:        markdownPath,
		AgentID:     "agent-a",
		WorkspaceID: "workspace-private",
		PeerID:      "user-juan",
	})
	results, err := restarted.Retrieve(ctx, "local markdown", 5)
	if err != nil {
		t.Fatalf("Retrieve after restart: %v", err)
	}
	if len(results) != 1 || !strings.Contains(results[0].Content, "local markdown and fast") {
		t.Fatalf("restart results = %+v, want persisted local markdown memory", results)
	}
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func assertFileContains(t *testing.T, path, want string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(raw), want) {
		t.Fatalf("%s missing %q:\n%s", path, want, raw)
	}
}

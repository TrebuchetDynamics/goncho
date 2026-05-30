package goncho

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	memory "github.com/TrebuchetDynamics/goncho/memory"
)

func TestLocalMarkdownMemoryPublicFacadePersistsThroughMemoryTools(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "memory.db")
	markdownPath := filepath.Join(t.TempDir(), "GONCHO_MEMORY.md")

	sqlite, err := memory.OpenSqlite(dbPath, 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer sqlite.Close(ctx)

	store := NewLocalMarkdownMemoryStore(sqlite.DB(), LocalMarkdownMemoryConfig{
		Path:           markdownPath,
		AgentID:        "agent-a",
		WorkspaceID:    "workspace-private",
		ObserverPeerID: "agent-a",
		PeerID:         "user-juan",
		SessionID:      "telegram:1",
	})
	status, err := store.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !status.Enabled || status.NetworkRequired || status.OllamaRequired {
		t.Fatalf("status = %+v, want enabled local-only markdown memory", status)
	}

	if _, err := NewStoreMemoryTool(store).Execute(ctx, json.RawMessage(`{
		"content":"Juan wants Goncho memory to be local markdown and fast.",
		"tags":["goncho","local"],
		"importance":0.9
	}`)); err != nil {
		t.Fatalf("store tool Execute: %v", err)
	}

	raw, err := NewRetrieveMemoryTool(store).Execute(ctx, json.RawMessage(`{"query":"local markdown","limit":5}`))
	if err != nil {
		t.Fatalf("retrieve tool Execute: %v", err)
	}
	var retrieved struct {
		Results []MemoryToolEntry `json:"results"`
		Count   int               `json:"count"`
	}
	if err := json.Unmarshal(raw, &retrieved); err != nil {
		t.Fatalf("unmarshal retrieve response: %v", err)
	}
	if retrieved.Count != 1 || len(retrieved.Results) != 1 || !strings.Contains(retrieved.Results[0].Content, "local markdown and fast") {
		t.Fatalf("retrieve response = %+v, want persisted local markdown memory", retrieved)
	}
	assertLocalMarkdownFileContains(t, markdownPath, "local markdown and fast")
}

func assertLocalMarkdownFileContains(t *testing.T, path, want string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(raw), want) {
		t.Fatalf("%s missing %q:\n%s", path, want, raw)
	}
}

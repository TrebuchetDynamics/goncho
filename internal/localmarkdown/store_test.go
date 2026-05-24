package localmarkdown_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

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

func TestLocalMarkdownMemoryStoreRanksRelevanceImportanceAndRecency(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	sqlite, err := memory.OpenSqlite(filepath.Join(dir, "memory.db"), 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	t.Cleanup(func() { _ = sqlite.Close(ctx) })
	store := localmarkdown.NewStore(sqlite.DB(), localmarkdown.Config{
		Path:           filepath.Join(dir, "GONCHO_MEMORY.md"),
		AgentID:        "agent-a",
		WorkspaceID:    "workspace-a",
		ObserverPeerID: "agent-a",
		PeerID:         "user-a",
		SessionID:      "telegram:42",
	})

	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	entries := []localmarkdown.Entry{
		{ID: "old-high", Content: "Telegram latency budget from an old incident.", Tags: []string{"latency"}, Importance: 0.95, CreatedAt: now.Add(-90 * 24 * time.Hour), UpdatedAt: now.Add(-90 * 24 * time.Hour)},
		{ID: "fresh-low", Content: "Telegram latency note from this morning.", Tags: []string{"latency"}, Importance: 0.2, CreatedAt: now.Add(-1 * time.Hour), UpdatedAt: now.Add(-1 * time.Hour)},
		{ID: "fresh-important", Content: "Telegram latency SLO must stay below eighty milliseconds.", Tags: []string{"latency", "slo"}, Importance: 0.8, CreatedAt: now.Add(-2 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)},
		{ID: "irrelevant", Content: "Theme preference is dark.", Tags: []string{"theme"}, Importance: 1.0, CreatedAt: now, UpdatedAt: now},
	}
	for _, entry := range entries {
		if err := store.Store(ctx, entry); err != nil {
			t.Fatalf("store %s: %v", entry.ID, err)
		}
	}

	results, err := store.Retrieve(ctx, "Telegram latency", 4)
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	var got []string
	for _, result := range results {
		got = append(got, result.ID)
	}
	want := []string{"fresh-important", "fresh-low", "old-high"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("retrieve ranked IDs = %v, want %v", got, want)
	}

	var irrelevantActive int
	if err := sqlite.DB().QueryRowContext(ctx, `SELECT active FROM goncho_memory_items WHERE memory_id = 'irrelevant'`).Scan(&irrelevantActive); err != nil {
		t.Fatalf("read irrelevant row: %v", err)
	}
	if irrelevantActive != 1 {
		t.Fatalf("irrelevant memory active = %d, want retained but not returned", irrelevantActive)
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

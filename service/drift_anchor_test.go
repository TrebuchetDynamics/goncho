package goncho

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/goncho/memory"
)

func TestGonchoGoalNegativeDriftAnchorWarnsBeforeRepeatedFailureE2E(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	store, err := memory.OpenSqlite(filepath.Join(dir, "memory.db"), 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(ctx)

	memoryStore := NewLocalMarkdownMemoryStore(store.DB(), LocalMarkdownMemoryConfig{
		Path:        filepath.Join(dir, "GONCHO_MEMORY.md"),
		AgentID:     "agent:mineru",
		WorkspaceID: "workspace-drift",
		PeerID:      "peer-drift",
		SessionID:   "session-drift",
	})
	storeTool := NewStoreMemoryTool(memoryStore)
	stored := executeMemoryTool(t, ctx, storeTool, `{"content":"Dead end: retrying stale Docker cache cleanup repeats a known failure; verify live container state first.","tags":["negative","dead-end","drift-anchor"],"importance":0.95}`)
	if stringField(t, stored, "id") == "" {
		t.Fatalf("store output = %+v, want memory id", stored)
	}

	detector := NewDriftAnchorDetector(memoryStore)
	warning, err := detector.Check(ctx, DriftAnchorCheckParams{
		Prompt: "Let's retry the stale Docker cache cleanup again before checking container state.",
		Limit:  5,
	})
	if err != nil {
		t.Fatalf("Check drift: %v", err)
	}
	if !warning.Warn || warning.Code != "negative_drift_anchor" {
		t.Fatalf("warning = %+v, want negative_drift_anchor", warning)
	}
	if warning.MatchedMemoryID == "" || warning.SimilarityScore <= 0 {
		t.Fatalf("warning evidence = %+v, want matched memory and positive score", warning)
	}
	if warning.Recommendation != "verify_live_state_before_repeating_failed_path" {
		t.Fatalf("recommendation = %q", warning.Recommendation)
	}

	safe, err := detector.Check(ctx, DriftAnchorCheckParams{
		Prompt: "Add documentation for the HTTP restart E2E report.",
		Limit:  5,
	})
	if err != nil {
		t.Fatalf("Check safe prompt: %v", err)
	}
	if safe.Warn {
		t.Fatalf("safe warning = %+v, want no drift warning", safe)
	}
}

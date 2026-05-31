package goncho

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/goncho/memory"
)

type scriptedDriftAnchorStore struct {
	byQuery map[string][]MemoryToolEntry
}

func (s scriptedDriftAnchorStore) Store(ctx context.Context, entry MemoryToolEntry) error { return nil }
func (s scriptedDriftAnchorStore) Retrieve(ctx context.Context, query string, limit int) ([]MemoryToolEntry, error) {
	entries := append([]MemoryToolEntry(nil), s.byQuery[query]...)
	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}
	return entries, nil
}
func (s scriptedDriftAnchorStore) Update(ctx context.Context, id string, content string) error {
	return nil
}
func (s scriptedDriftAnchorStore) Forget(ctx context.Context, id string) error { return nil }

func TestDriftAnchorChecksNegativeFallbackWhenDeadEndQueryHasNoAnchorMatch(t *testing.T) {
	detector := NewDriftAnchorDetector(scriptedDriftAnchorStore{byQuery: map[string][]MemoryToolEntry{
		"dead-end": {
			{ID: "unrelated-dead-end", Content: "Dead end: unrelated package manager retry failed.", Tags: []string{"dead-end"}},
		},
		"negative": {
			{ID: "matching-negative", Content: "Known failure: stale Docker cache cleanup repeats unless live container state is verified first.", Tags: []string{"negative", "drift-anchor"}},
		},
	}})

	warning, err := detector.Check(context.Background(), DriftAnchorCheckParams{
		Prompt: "Retry the stale Docker cache cleanup again before checking live container state.",
		Limit:  5,
	})
	if err != nil {
		t.Fatalf("Check drift: %v", err)
	}
	if !warning.Warn || warning.MatchedMemoryID != "matching-negative" {
		t.Fatalf("warning = %+v, want matching negative fallback anchor", warning)
	}
}

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

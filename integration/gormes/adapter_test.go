package gormes_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	gormes "github.com/TrebuchetDynamics/goncho/integration/gormes"
)

func TestOpenRuntimeBuildsGormesReadyServiceAndTools(t *testing.T) {
	ctx := context.Background()
	runtime, err := gormes.Open(ctx, gormes.Config{
		DatabasePath: filepath.Join(t.TempDir(), "goncho.db"),
		WorkspaceID:  "gormes-test",
		ObserverID:   "gormes",
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer runtime.Close(ctx)

	if runtime.Service == nil || runtime.ContextTool == nil || runtime.SearchTool == nil || runtime.RememberTool == nil || runtime.ReviewTool == nil || runtime.HandoffTool == nil {
		t.Fatalf("runtime tools not fully initialized: %+v", runtime)
	}
	status := runtime.Status()
	if !status.Ready || status.WorkspaceID != "gormes-test" || status.ObserverID != "gormes" || status.DatabasePath == "" {
		t.Fatalf("status = %+v, want ready gormes-test/gormes with db path", status)
	}
	wantTools := []string{"goncho_context", "goncho_search", "goncho_remember", "goncho_review", "goncho_handoff"}
	for _, want := range wantTools {
		if !contains(status.ToolNames, want) {
			t.Fatalf("tool names = %#v, missing %s", status.ToolNames, want)
		}
	}

	remembered, err := runtime.RememberTool.Execute(ctx, json.RawMessage(`{"peer_id":"user-1","session_key":"session-1","content":"User prefers local SQLite memory."}`))
	if err != nil {
		t.Fatalf("remember execute: %v", err)
	}
	if len(remembered) == 0 {
		t.Fatalf("remember returned empty payload")
	}
	contextPayload, err := runtime.ContextTool.Execute(ctx, json.RawMessage(`{"peer_id":"user-1","session_key":"session-1","query":"database preference","max_tokens":500}`))
	if err != nil {
		t.Fatalf("context execute: %v", err)
	}
	if len(contextPayload) == 0 {
		t.Fatalf("context returned empty payload")
	}
}

func TestOpenRuntimeUsesDeploySafeDefaults(t *testing.T) {
	ctx := context.Background()
	runtime, err := gormes.Open(ctx, gormes.Config{DatabasePath: filepath.Join(t.TempDir(), "goncho.db")})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer runtime.Close(ctx)

	status := runtime.Status()
	if status.WorkspaceID != "gormes" || status.ObserverID != "gormes" {
		t.Fatalf("status = %+v, want default gormes workspace/observer", status)
	}
}

func TestOpenRuntimeRejectsEmptyDatabasePath(t *testing.T) {
	_, err := gormes.Open(context.Background(), gormes.Config{})
	if err == nil {
		t.Fatalf("Open with empty database path succeeded, want deploy-safe error")
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

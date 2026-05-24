package gormes_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
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

	if runtime.Service == nil || runtime.ContextTool == nil || runtime.SearchTool == nil || runtime.RecallTool == nil || runtime.RememberTool == nil || runtime.ReviewTool == nil || runtime.HandoffTool == nil {
		t.Fatalf("runtime tools not fully initialized: %+v", runtime)
	}
	status := runtime.Status()
	if !status.Ready || status.WorkspaceID != "gormes-test" || status.ObserverID != "gormes" || status.DatabasePath == "" {
		t.Fatalf("status = %+v, want ready gormes-test/gormes with db path", status)
	}
	wantTools := []string{"goncho_context", "goncho_search", "goncho_recall", "goncho_remember", "goncho_review", "goncho_handoff"}
	for _, want := range wantTools {
		if !contains(status.ToolNames, want) {
			t.Fatalf("tool names = %#v, missing %s", status.ToolNames, want)
		}
	}
	if len(status.ToolSpecs) != len(status.ToolNames) {
		t.Fatalf("tool specs len = %d, tool names len = %d", len(status.ToolSpecs), len(status.ToolNames))
	}
	recallSpec, ok := operationSpecByName(status.ToolSpecs, "goncho_recall")
	if !ok {
		t.Fatalf("tool specs = %#v, missing goncho_recall", status.ToolSpecs)
	}
	if recallSpec.Mutating || !recallSpec.Idempotent || recallSpec.AuditKind != "memory" {
		t.Fatalf("goncho_recall spec = %+v, want read-only idempotent memory tool", recallSpec)
	}
	var recallSchema struct {
		Properties map[string]any `json:"properties"`
	}
	if err := json.Unmarshal(recallSpec.Schema, &recallSchema); err != nil {
		t.Fatalf("recall schema json: %v", err)
	}
	if _, ok := recallSchema.Properties["compact"]; !ok {
		t.Fatalf("goncho_recall schema properties = %#v, missing compact", recallSchema.Properties)
	}
	statusJSON, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("marshal status: %v", err)
	}
	if !strings.Contains(string(statusJSON), "tool_specs") || !strings.Contains(string(statusJSON), "compact") {
		t.Fatalf("status json = %s, want tool specs with recall compact schema", statusJSON)
	}
	if strings.Contains(string(statusJSON), "ToolDescriptor") || strings.Contains(string(statusJSON), "Mutating") {
		t.Fatalf("status json = %s, leaked internal OperationSpec field names", statusJSON)
	}
	var statusDocument struct {
		ToolSpecs []map[string]any `json:"tool_specs"`
	}
	if err := json.Unmarshal(statusJSON, &statusDocument); err != nil {
		t.Fatalf("status json document: %v", err)
	}
	jsonRecallSpec, ok := statusToolSpecByName(statusDocument.ToolSpecs, "goncho_recall")
	if !ok {
		t.Fatalf("status json tool specs = %#v, missing goncho_recall", statusDocument.ToolSpecs)
	}
	for _, key := range []string{"name", "description", "schema", "mutating", "idempotent", "prompt_safe", "trust_class", "audit_kind"} {
		if _, ok := jsonRecallSpec[key]; !ok {
			t.Fatalf("status json goncho_recall spec = %#v, missing %q", jsonRecallSpec, key)
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
	recallPayload, err := runtime.RecallTool.Execute(ctx, json.RawMessage(`{"peer_id":"user-1","session_key":"session-1","query":"local SQLite memory","limit":1}`))
	if err != nil {
		t.Fatalf("recall execute: %v", err)
	}
	if len(recallPayload) == 0 {
		t.Fatalf("recall returned empty payload")
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

func TestOpenRuntimeDerivesProfilePathsFromProfilesDirectory(t *testing.T) {
	ctx := context.Background()
	profilesDir := filepath.Join(t.TempDir(), ".gormes", "profiles")
	runtime, err := gormes.Open(ctx, gormes.Config{
		ProfilesDirectory: profilesDir,
		ProfileID:         "mineru",
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer runtime.Close(ctx)

	status := runtime.Status()
	wantProfileDir := filepath.Join(profilesDir, "mineru")
	if status.ProfileID != "mineru" {
		t.Fatalf("profile_id = %q, want mineru", status.ProfileID)
	}
	if status.ProfilesDirectory != profilesDir {
		t.Fatalf("profiles_directory = %q, want %q", status.ProfilesDirectory, profilesDir)
	}
	if status.ProfileDirectory != wantProfileDir {
		t.Fatalf("profile_directory = %q, want %q", status.ProfileDirectory, wantProfileDir)
	}
	if status.DatabasePath != filepath.Join(wantProfileDir, "goncho.db") {
		t.Fatalf("database_path = %q", status.DatabasePath)
	}
	if status.MemoryMarkdownPath != filepath.Join(wantProfileDir, "GONCHO_MEMORY.md") {
		t.Fatalf("memory_markdown_path = %q", status.MemoryMarkdownPath)
	}
}

func TestOpenRuntimeRejectsEmptyDatabasePath(t *testing.T) {
	_, err := gormes.Open(context.Background(), gormes.Config{})
	if err == nil {
		t.Fatalf("Open with empty database path succeeded, want deploy-safe error")
	}
}

func TestOpenRuntimeRejectsUnsafeProfileIDForProfilesDirectory(t *testing.T) {
	_, err := gormes.Open(context.Background(), gormes.Config{ProfilesDirectory: t.TempDir(), ProfileID: "../mineru"})
	if err == nil {
		t.Fatalf("Open with unsafe profile id succeeded, want error")
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

func operationSpecByName(specs []gormes.StatusToolSpec, want string) (gormes.StatusToolSpec, bool) {
	for _, spec := range specs {
		if spec.Name == want {
			return spec, true
		}
	}
	return gormes.StatusToolSpec{}, false
}

func statusToolSpecByName(specs []map[string]any, want string) (map[string]any, bool) {
	for _, spec := range specs {
		if spec["name"] == want {
			return spec, true
		}
	}
	return nil, false
}

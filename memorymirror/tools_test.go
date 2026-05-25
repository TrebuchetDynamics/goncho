package memorymirror

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/goncho/memory"
	goncho "github.com/TrebuchetDynamics/goncho/service"
	"github.com/TrebuchetDynamics/goncho/toolmeta"
)

func TestBroadMemoryCompatibleToolRegistryExecutesCoreAliases(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newMemoryMirrorTestService(t)
	defer cleanup()

	peer := "peer-memorymirror"
	session := "session-memorymirror"
	if _, err := svc.CreateMessages(ctx, goncho.CreateMessagesParams{SessionKey: session, Messages: []goncho.CreateMessage{{Peer: peer, Role: "user", Content: "Need auth middleware timeline evidence."}}}); err != nil {
		t.Fatalf("CreateMessages: %v", err)
	}
	observed, err := svc.Observe(ctx, goncho.ObservationParams{Kind: goncho.ObservationKindToolCall, PeerID: peer, SessionKey: session, Input: "rg jose middleware"})
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}

	tools := NewToolRegistry(svc, ToolRegistryOptions{DefaultPeerID: peer, DefaultSessionKey: session})
	for _, want := range []string{"memory_save", "memory_smart_search", "memory_recall", "memory_timeline", "memory_audit"} {
		if _, ok := findTool(tools, want); !ok {
			t.Fatalf("tool registry missing %s; names=%v", want, toolNames(tools))
		}
	}

	saved := executeMemoryMirrorTool(t, ctx, tools, "memory_save", map[string]any{
		"content":  "Agentmemory-compatible auth architecture uses jose middleware.",
		"type":     "architecture",
		"concepts": "auth,jose,middleware",
		"files":    "src/middleware/auth.ts",
	})
	if saved["success"] != true || saved["backend"] != "goncho" || saved["tool"] != "memory_save" {
		t.Fatalf("memory_save output = %+v", saved)
	}
	if saved["id"] == nil || saved["id"] == float64(0) {
		t.Fatalf("memory_save id = %#v", saved["id"])
	}

	searched := executeMemoryMirrorTool(t, ctx, tools, "memory_smart_search", map[string]any{
		"query": "jose auth middleware",
		"limit": 5,
	})
	if searched["tool"] != "memory_smart_search" || int(searched["count"].(float64)) < 1 {
		t.Fatalf("memory_smart_search output = %+v", searched)
	}
	if searched["retrieval"] != "goncho_search" {
		t.Fatalf("memory_smart_search retrieval = %#v", searched["retrieval"])
	}

	recalled := executeMemoryMirrorTool(t, ctx, tools, "memory_recall", map[string]any{
		"query":        "What auth middleware did we choose?",
		"limit":        5,
		"format":       "compact",
		"token_budget": 2000,
	})
	if recalled["tool"] != "memory_recall" || recalled["retrieval"] != "goncho_recall" {
		t.Fatalf("memory_recall output = %+v", recalled)
	}
	if int(recalled["selected_count"].(float64)) < 1 {
		t.Fatalf("memory_recall selected_count = %#v", recalled["selected_count"])
	}

	timeline := executeMemoryMirrorTool(t, ctx, tools, "memory_timeline", map[string]any{"session_id": session})
	if timeline["tool"] != "memory_timeline" || timeline["retrieval"] != "goncho_viewer_timeline" || int(timeline["event_count"].(float64)) < 2 {
		t.Fatalf("memory_timeline output = %+v", timeline)
	}

	audit := executeMemoryMirrorTool(t, ctx, tools, "memory_audit", map[string]any{"target_id": observed.Observation.ID})
	if audit["tool"] != "memory_audit" || audit["retrieval"] != "goncho_audit_trail" || int(audit["count"].(float64)) != 1 {
		t.Fatalf("memory_audit output = %+v", audit)
	}
}

func TestCompatibilityCatalogDocumentsRegisteredSafeAliases(t *testing.T) {
	svc, cleanup := newMemoryMirrorTestService(t)
	defer cleanup()

	catalog := CompatibilityCatalog()
	for _, want := range []string{"memory_save", "memory_smart_search", "memory_recall", "memory_profile", "memory_timeline", "memory_audit"} {
		entry, ok := catalog.CompatTool(want)
		if !ok {
			t.Fatalf("compat catalog missing %s", want)
		}
		if entry.Status != PortDelivered || entry.RegisteredName != want {
			t.Fatalf("catalog[%s] = %+v, want delivered registered alias", want, entry)
		}
	}

	manifest := ArchitectureManifest()
	for _, entry := range catalog.Tools {
		if _, ok := manifest.Tool(entry.Name); !ok {
			t.Fatalf("catalog tool %s missing from architecture manifest", entry.Name)
		}
		if entry.DefaultEnabled && !strings.HasPrefix(entry.GonchoSeam, "service.") {
			t.Fatalf("default-enabled tool %s seam = %q, want public service seam", entry.Name, entry.GonchoSeam)
		}
	}
	for _, tool := range NewToolRegistry(svc, ToolRegistryOptions{}) {
		entry, ok := catalog.CompatTool(tool.Name())
		if !ok {
			t.Fatalf("registered tool %s missing from compat catalog", tool.Name())
		}
		if entry.Status != PortDelivered {
			t.Fatalf("registered tool %s status = %q, want delivered", tool.Name(), entry.Status)
		}
	}
}

func TestBroadMemoryCompatibleToolsExposeUpstreamSchemasAndSpecs(t *testing.T) {
	svc, cleanup := newMemoryMirrorTestService(t)
	defer cleanup()

	tools := NewToolRegistry(svc, ToolRegistryOptions{})
	for _, name := range []string{"memory_save", "memory_smart_search", "memory_recall", "memory_profile", "memory_timeline", "memory_audit"} {
		tool, ok := findTool(tools, name)
		if !ok {
			t.Fatalf("missing %s", name)
		}
		specTool, ok := tool.(toolmeta.Spec)
		if !ok {
			t.Fatalf("%s does not expose OperationSpec", name)
		}
		spec := specTool.Spec()
		if spec.Name != name || string(spec.Schema) == "" || !spec.PromptSafe {
			t.Fatalf("%s spec = %+v", name, spec)
		}
		if name == "memory_save" && !spec.Mutating {
			t.Fatalf("memory_save must be marked mutating")
		}
		if (name == "memory_timeline" || name == "memory_audit") && spec.AuditKind != "memory" {
			t.Fatalf("%s audit kind = %q, want memory", name, spec.AuditKind)
		}
		if name != "memory_save" && spec.Mutating {
			t.Fatalf("%s must be non-mutating", name)
		}
	}

	manifest := ArchitectureManifest()
	for _, tool := range tools {
		if _, ok := manifest.Tool(tool.Name()); !ok {
			t.Fatalf("registry tool %s is not represented in architecture manifest", tool.Name())
		}
	}
}

func newMemoryMirrorTestService(t *testing.T) (*goncho.Service, func()) {
	t.Helper()
	store, err := memory.OpenSqlite(t.TempDir()+"/memorymirror.db", 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	if err := goncho.RunMigrations(store.DB()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	svc := goncho.NewService(store.DB(), goncho.Config{WorkspaceID: "memorymirror-test", ObserverPeerID: "memorymirror-adapter"}, nil)
	return svc, func() { _ = store.Close(context.Background()) }
}

func findTool(tools []toolmeta.Tool, name string) (toolmeta.Tool, bool) {
	for _, tool := range tools {
		if tool.Name() == name {
			return tool, true
		}
	}
	return nil, false
}

func toolNames(tools []toolmeta.Tool) []string {
	out := make([]string, 0, len(tools))
	for _, tool := range tools {
		out = append(out, tool.Name())
	}
	slices.Sort(out)
	return out
}

func executeMemoryMirrorTool(t *testing.T, ctx context.Context, tools []toolmeta.Tool, name string, args map[string]any) map[string]any {
	t.Helper()
	tool, ok := findTool(tools, name)
	if !ok {
		t.Fatalf("missing tool %s", name)
	}
	rawArgs, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Marshal args: %v", err)
	}
	raw, err := tool.Execute(ctx, rawArgs)
	if err != nil {
		t.Fatalf("%s Execute: %v", name, err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("%s output JSON: %v\n%s", name, err, raw)
	}
	return out
}

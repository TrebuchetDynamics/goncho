package memorymirror

import (
	"context"
	"encoding/json"
	"slices"
	"testing"

	"github.com/TrebuchetDynamics/goncho/memory"
	goncho "github.com/TrebuchetDynamics/goncho/service"
	"github.com/TrebuchetDynamics/goncho/toolmeta"
)

func TestBroadMemoryCompatibleToolRegistryExecutesCoreAliases(t *testing.T) {
	ctx := context.Background()
	svc, cleanup := newMemoryMirrorTestService(t)
	defer cleanup()

	tools := NewToolRegistry(svc, ToolRegistryOptions{DefaultPeerID: "peer-memorymirror", DefaultSessionKey: "session-memorymirror"})
	for _, want := range []string{"memory_save", "memory_smart_search", "memory_recall"} {
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
}

func TestBroadMemoryCompatibleToolsExposeUpstreamSchemasAndSpecs(t *testing.T) {
	svc, cleanup := newMemoryMirrorTestService(t)
	defer cleanup()

	tools := NewToolRegistry(svc, ToolRegistryOptions{})
	for _, name := range []string{"memory_save", "memory_smart_search", "memory_recall", "memory_profile"} {
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

package agentmemory

import "testing"

func TestArchitectureMirrorTracksCurrentAgentMemoryUpstream(t *testing.T) {
	manifest := ArchitectureManifest()
	if manifest.Source.Repository != "https://github.com/rohitg00/agentmemory" {
		t.Fatalf("source repository = %q", manifest.Source.Repository)
	}
	if manifest.Source.Commit != "355124141625ccc0d740ae08ddaaf77fe2c165ae" {
		t.Fatalf("source commit = %q", manifest.Source.Commit)
	}
	if len(manifest.MemoryTiers) != 4 {
		t.Fatalf("memory tiers = %d, want 4", len(manifest.MemoryTiers))
	}
	for _, want := range []string{"working", "episodic", "semantic", "procedural"} {
		if _, ok := manifest.MemoryTier(want); !ok {
			t.Fatalf("missing memory tier %q", want)
		}
	}
	for _, want := range []string{"bm25", "vector", "graph"} {
		stream, ok := manifest.RetrievalStream(want)
		if !ok {
			t.Fatalf("missing retrieval stream %q", want)
		}
		if stream.Fusion != "reciprocal_rank_fusion" {
			t.Fatalf("stream %q fusion = %q", want, stream.Fusion)
		}
	}
	for _, want := range []string{"PostToolUse", "SessionEnd", "PreCompact", "SessionStart"} {
		if _, ok := manifest.Hook(want); !ok {
			t.Fatalf("missing hook %q", want)
		}
	}
	if got, want := len(manifest.Tools), 53; got != want {
		t.Fatalf("tools = %d, want %d", got, want)
	}
}

func TestPortMatrixMapsDeliveredCoreToolsToGonchoSeams(t *testing.T) {
	manifest := ArchitectureManifest()
	cases := map[string]string{
		"memory_save":              "service.Service.Conclude",
		"memory_recall":            "service.Service.Recall",
		"memory_smart_search":      "service.Service.Search",
		"memory_profile":           "service.Service.Profile",
		"memory_export":            "memory.GonchoMemoryV1Document",
		"memory_audit":             "service.AuditTrail",
		"memory_governance_delete": "service.Service.Conclude(DeleteID)",
	}
	for tool, wantSeam := range cases {
		capability, ok := manifest.Tool(tool)
		if !ok {
			t.Fatalf("missing tool %q", tool)
		}
		if capability.Status != PortDelivered {
			t.Fatalf("%s status = %q, want %q", tool, capability.Status, PortDelivered)
		}
		if capability.GonchoSeam != wantSeam {
			t.Fatalf("%s seam = %q, want %q", tool, capability.GonchoSeam, wantSeam)
		}
	}
}

func TestPortMatrixDoesNotOverclaimUnsupportedExtendedAgentMemoryTools(t *testing.T) {
	manifest := ArchitectureManifest()
	for _, tool := range []string{"memory_vision_search", "memory_mesh_sync", "memory_slot_delete"} {
		capability, ok := manifest.Tool(tool)
		if !ok {
			t.Fatalf("missing tool %q", tool)
		}
		if capability.Status == PortDelivered {
			t.Fatalf("%s is marked delivered without a Goncho implementation", tool)
		}
		if capability.Residual == "" {
			t.Fatalf("%s residual is empty", tool)
		}
	}
}

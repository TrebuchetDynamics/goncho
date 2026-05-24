package memorymirror

import "testing"

func TestImplementationBacklogPrioritizesBenchmarkRelevantFeatures(t *testing.T) {
	items := ImplementationBacklog()
	if len(items) < 8 {
		t.Fatalf("backlog has %d items, want at least 8", len(items))
	}
	assertBacklogItem(t, items, "local_vector_embeddings", PriorityP0, PortDeferred)
	assertBacklogItem(t, items, "automatic_hook_capture", PriorityP0, PortPartial)
	assertBacklogItem(t, items, "query_expansion_synonyms", PriorityP0, PortPartial)
	assertBacklogItem(t, items, "memory_resources_prompts", PriorityP1, PortDeferred)
	assertBacklogItem(t, items, "slot_memory", PriorityP1, PortDeferred)
	assertBacklogItem(t, items, "action_graph_leases_signals", PriorityP2, PortPartial)
}

func TestImplementationBacklogHasNoUpstreamProjectNameInPublicIDs(t *testing.T) {
	for _, item := range ImplementationBacklog() {
		if item.ID == "agentmemory" || item.PublicName == "agentmemory" {
			t.Fatalf("backlog item exposes upstream project name: %+v", item)
		}
		if item.Rationale == "" || item.SmallestSlice == "" || item.GonchoSeam == "" {
			t.Fatalf("backlog item lacks implementation guidance: %+v", item)
		}
	}
}

func TestBacklogByPriorityReturnsDefensiveCopy(t *testing.T) {
	p0 := BacklogByPriority(PriorityP0)
	if len(p0) == 0 {
		t.Fatalf("no P0 backlog items")
	}
	p0[0].ID = "mutated"
	again := BacklogByPriority(PriorityP0)
	if again[0].ID == "mutated" {
		t.Fatalf("BacklogByPriority returned mutable backing storage")
	}
}

func assertBacklogItem(t *testing.T, items []BacklogItem, id string, priority BacklogPriority, status PortStatus) {
	t.Helper()
	for _, item := range items {
		if item.ID == id {
			if item.Priority != priority || item.CurrentStatus != status {
				t.Fatalf("%s = %+v, want priority %s status %s", id, item, priority, status)
			}
			return
		}
	}
	t.Fatalf("backlog missing %s", id)
}

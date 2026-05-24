package memorymirror

import "slices"

type BacklogPriority string

const (
	PriorityP0 BacklogPriority = "P0"
	PriorityP1 BacklogPriority = "P1"
	PriorityP2 BacklogPriority = "P2"
	PriorityP3 BacklogPriority = "P3"
)

type BacklogItem struct {
	ID            string          `json:"id"`
	PublicName    string          `json:"public_name"`
	Priority      BacklogPriority `json:"priority"`
	CurrentStatus PortStatus      `json:"current_status"`
	GonchoSeam    string          `json:"goncho_seam"`
	Rationale     string          `json:"rationale"`
	SmallestSlice string          `json:"smallest_slice"`
}

func ImplementationBacklog() []BacklogItem {
	return slices.Clone(memoryMirrorBacklog)
}

func BacklogByPriority(priority BacklogPriority) []BacklogItem {
	var out []BacklogItem
	for _, item := range memoryMirrorBacklog {
		if item.Priority == priority {
			out = append(out, item)
		}
	}
	return out
}

var memoryMirrorBacklog = []BacklogItem{
	{
		ID:            "local_vector_embeddings",
		PublicName:    "Local vector embeddings",
		Priority:      PriorityP0,
		CurrentStatus: PortDeferred,
		GonchoSeam:    "service.Search/Recall candidate generation",
		Rationale:     "Closest upstream retrieval gap: local dense vectors can improve semantic matches and explain the all-MiniLM benchmark target without requiring hosted infrastructure.",
		SmallestSlice: "Add optional embedded-vector store interface plus deterministic fake-vector test proving vector candidates participate in RRF after lexical candidates.",
	},
	{
		ID:            "automatic_hook_capture",
		PublicName:    "Automatic hook capture",
		Priority:      PriorityP0,
		CurrentStatus: PortPartial,
		GonchoSeam:    "plugins write queue + service.Observe",
		Rationale:     "Upstream's strongest UX feature is zero-manual capture from tool/session lifecycle. Goncho has observation storage but hosts still wire most lifecycle events manually.",
		SmallestSlice: "Expose a host-neutral capture event adapter that maps PostToolUse/UserPromptSubmit/SessionEnd payloads to Observe/CreateMessages/session summary with privacy filtering.",
	},
	{
		ID:            "query_expansion_synonyms",
		PublicName:    "Query expansion and synonym routing",
		Priority:      PriorityP0,
		CurrentStatus: PortPartial,
		GonchoSeam:    "service.Search lexical/fact intent scoring",
		Rationale:     "Improves recall without LLM judges or benchmark-specific gold hacks; aligns with upstream smart search and current LongMemEval residual failure research.",
		SmallestSlice: "Add a small transparent synonym/alias expansion trace to Search/Recall and prove one fixture moves from miss to hit with expansion provenance.",
	},
	{
		ID:            "memory_resources_prompts",
		PublicName:    "Memory resources and prompts",
		Priority:      PriorityP1,
		CurrentStatus: PortDeferred,
		GonchoSeam:    "toolmeta + host adapter resource registry",
		Rationale:     "Upstream exposes status/profile/latest/graph resources and recall/handoff prompts. Goncho has data; it lacks a neutral discovery surface for hosts.",
		SmallestSlice: "Return static resource/prompt descriptors plus generated status/profile/latest payloads from public Go APIs, no MCP server required.",
	},
	{
		ID:            "slot_memory",
		PublicName:    "Slot memory",
		Priority:      PriorityP1,
		CurrentStatus: PortDeferred,
		GonchoSeam:    "memory scoped key/value plus review/audit",
		Rationale:     "Slots are useful for durable named facts/preferences without running a full search; current matrix marks all slot tools deferred.",
		SmallestSlice: "Implement create/get/list/append/replace/delete for scoped slots with audit rows and tests for profile/session isolation.",
	},
	{
		ID:            "consolidation_worker",
		PublicName:    "Four-tier consolidation worker",
		Priority:      PriorityP1,
		CurrentStatus: PortPartial,
		GonchoSeam:    "dream scheduler + session summaries + memory annotations",
		Rationale:     "Goncho has pieces of working/episodic/semantic/procedural memory but no single auditable consolidation pass comparable to upstream's pipeline.",
		SmallestSlice: "Add an explicit local consolidation command that reads observations/session summaries and emits reviewed semantic/procedural candidates with provenance.",
	},
	{
		ID:            "action_graph_leases_signals",
		PublicName:    "Action graph, leases, and signals",
		Priority:      PriorityP2,
		CurrentStatus: PortPartial,
		GonchoSeam:    "review/work-intent queues + future server mode",
		Rationale:     "Useful for multi-agent coordination, but less important than retrieval quality and capture; distributed guarantees belong behind server/team mode.",
		SmallestSlice: "Start local-only: action create/update/frontier with dependency blocking and audit, then add leases/signals only under explicit server-mode governance.",
	},
	{
		ID:            "git_snapshots",
		PublicName:    "Git-versioned memory snapshots",
		Priority:      PriorityP2,
		CurrentStatus: PortAdapter,
		GonchoSeam:    "memory.GonchoMemoryV1Document + host git adapter",
		Rationale:     "Snapshot/diff/rollback is valuable for auditability, but host-owned git operations avoid dangerous implicit repository mutation in the core library.",
		SmallestSlice: "Expose deterministic export manifests with checksum/diff metadata; leave git commit/rollback to an adapter that requires explicit owner action.",
	},
	{
		ID:            "vision_search",
		PublicName:    "Vision/image search",
		Priority:      PriorityP3,
		CurrentStatus: PortDeferred,
		GonchoSeam:    "optional media reference index",
		Rationale:     "Upstream supports CLIP-style image search, but Goncho's current benchmark and host needs are text-memory first.",
		SmallestSlice: "Store image references and metadata checksums first; defer embeddings until a local Go-friendly provider is selected.",
	},
}

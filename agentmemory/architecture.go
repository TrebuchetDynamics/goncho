// Package agentmemory records Goncho's Go-native architecture mirror of the
// upstream agentmemory system without importing its TypeScript runtime.
package agentmemory

import "strings"

const (
	PortDelivered PortStatus = "delivered"
	PortPartial   PortStatus = "partial"
	PortAdapter   PortStatus = "adapter"
	PortDeferred  PortStatus = "deferred"
	PortExcluded  PortStatus = "owned_excluded"
)

type PortStatus string

type Source struct {
	Repository string   `json:"repository"`
	Commit     string   `json:"commit"`
	Refs       []string `json:"refs"`
}

type Architecture struct {
	Source           Source            `json:"source"`
	Pipeline         []PipelineStage   `json:"pipeline"`
	MemoryTiers      []MemoryTier      `json:"memory_tiers"`
	RetrievalStreams []RetrievalStream `json:"retrieval_streams"`
	Hooks            []Hook            `json:"hooks"`
	Tools            []Capability      `json:"tools"`
}

type PipelineStage struct {
	Name       string     `json:"name"`
	Upstream   string     `json:"upstream"`
	GonchoSeam string     `json:"goncho_seam"`
	Status     PortStatus `json:"status"`
	Residual   string     `json:"residual,omitempty"`
}

type MemoryTier struct {
	Name       string     `json:"name"`
	Upstream   string     `json:"upstream"`
	GonchoSeam string     `json:"goncho_seam"`
	Status     PortStatus `json:"status"`
}

type RetrievalStream struct {
	Name       string     `json:"name"`
	Upstream   string     `json:"upstream"`
	Fusion     string     `json:"fusion"`
	GonchoSeam string     `json:"goncho_seam"`
	Status     PortStatus `json:"status"`
	Residual   string     `json:"residual,omitempty"`
}

type Hook struct {
	Name       string     `json:"name"`
	Captures   string     `json:"captures"`
	GonchoSeam string     `json:"goncho_seam"`
	Status     PortStatus `json:"status"`
	Residual   string     `json:"residual,omitempty"`
}

type Capability struct {
	Name       string     `json:"name"`
	Kind       string     `json:"kind"`
	GonchoSeam string     `json:"goncho_seam"`
	Status     PortStatus `json:"status"`
	Residual   string     `json:"residual,omitempty"`
}

func ArchitectureManifest() Architecture {
	return Architecture{
		Source: Source{
			Repository: "https://github.com/rohitg00/agentmemory",
			Commit:     "355124141625ccc0d740ae08ddaaf77fe2c165ae",
			Refs: []string{
				"README.md#how-it-works",
				"README.md#mcp-server",
				"src/mcp/tools-registry.ts",
			},
		},
		Pipeline: []PipelineStage{
			stage("capture", "hooks capture prompts, tools, failures, lifecycle", "plugins write queue + service.Observe", PortPartial, "host-specific hook installers are outside the Goncho library"),
			stage("dedup_privacy", "SHA-256 dedup then privacy filtering", "internal/observationlog + internal/fileimport quarantine", PortPartial, "secret redaction is local/conservative, not a byte-for-byte port"),
			stage("raw_observation", "store raw observation", "service.Observe/ListObservations/AuditTrail", PortDelivered, ""),
			stage("compression", "LLM compresses observations into facts, concepts, narrative", "service session summaries + memory annotations", PortPartial, "Goncho avoids mandatory provider calls in the core library"),
			stage("index", "BM25 + vector + graph indexes", "service.Search/Recall + annotation graph", PortPartial, "vector embeddings remain intentionally deferred"),
			stage("context_injection", "SessionStart loads profile, hybrid search, token budget", "service.Context/RecallProjector", PortDelivered, ""),
		},
		MemoryTiers: []MemoryTier{
			tier("working", "raw observations from tool use", "service.Observe + observationlog", PortDelivered),
			tier("episodic", "compressed session summaries", "service session summaries", PortDelivered),
			tier("semantic", "extracted facts and patterns", "service.Conclude + memoryannotations", PortDelivered),
			tier("procedural", "workflows and decision patterns", "skill proposals + review queue + actions", PortPartial),
		},
		RetrievalStreams: []RetrievalStream{
			stream("bm25", "stemmed keyword matching with synonym expansion", "reciprocal_rank_fusion", "service.Search lexical scoring", PortDelivered, ""),
			stream("vector", "cosine similarity over dense embeddings", "reciprocal_rank_fusion", "not enabled in Goncho core", PortDeferred, "local-first Go core currently avoids embedding-provider dependencies"),
			stream("graph", "entity matching and knowledge graph traversal", "reciprocal_rank_fusion", "service.Recall graph expansion + annotations", PortDelivered, ""),
		},
		Hooks: []Hook{
			hook("SessionStart", "project path and session id", "service.Context", PortAdapter, "host supplies lifecycle events"),
			hook("UserPromptSubmit", "user prompts", "service.CreateMessages/Observe", PortAdapter, "host supplies prompts"),
			hook("PreToolUse", "file access patterns and enriched context", "service.Context + review warnings", PortPartial, "Goncho provides warnings, not host hook installation"),
			hook("PostToolUse", "tool name, input, output", "service.Observe + AuditTrail", PortPartial, "host must forward tool events"),
			hook("PostToolUseFailure", "error context", "service.Observe + review queue", PortPartial, "host must forward failure events"),
			hook("PreCompact", "memory reinjection before compaction", "service.Context", PortDelivered, ""),
			hook("SubagentStart", "sub-agent lifecycle start", "dynamicagents registry", PortPartial, "host must forward subagent events"),
			hook("SubagentStop", "sub-agent lifecycle stop", "dynamicagents registry", PortPartial, "host must forward subagent events"),
			hook("Stop", "end-of-session summary", "service session summaries", PortPartial, "summary generation is host/provider mediated"),
			hook("SessionEnd", "session complete marker", "service session summaries + audit", PortPartial, "host must forward lifecycle events"),
		},
		Tools: toolManifest(),
	}
}

func (a Architecture) MemoryTier(name string) (MemoryTier, bool) {
	needle := normalize(name)
	for _, tier := range a.MemoryTiers {
		if normalize(tier.Name) == needle {
			return tier, true
		}
	}
	return MemoryTier{}, false
}

func (a Architecture) RetrievalStream(name string) (RetrievalStream, bool) {
	needle := normalize(name)
	for _, stream := range a.RetrievalStreams {
		if normalize(stream.Name) == needle {
			return stream, true
		}
	}
	return RetrievalStream{}, false
}

func (a Architecture) Hook(name string) (Hook, bool) {
	needle := normalize(name)
	for _, hook := range a.Hooks {
		if normalize(hook.Name) == needle {
			return hook, true
		}
	}
	return Hook{}, false
}

func (a Architecture) Tool(name string) (Capability, bool) {
	needle := normalize(name)
	for _, tool := range a.Tools {
		if normalize(tool.Name) == needle {
			return tool, true
		}
	}
	return Capability{}, false
}

func stage(name, upstream, seam string, status PortStatus, residual string) PipelineStage {
	return PipelineStage{Name: name, Upstream: upstream, GonchoSeam: seam, Status: status, Residual: residual}
}

func tier(name, upstream, seam string, status PortStatus) MemoryTier {
	return MemoryTier{Name: name, Upstream: upstream, GonchoSeam: seam, Status: status}
}

func stream(name, upstream, fusion, seam string, status PortStatus, residual string) RetrievalStream {
	return RetrievalStream{Name: name, Upstream: upstream, Fusion: fusion, GonchoSeam: seam, Status: status, Residual: residual}
}

func hook(name, captures, seam string, status PortStatus, residual string) Hook {
	return Hook{Name: name, Captures: captures, GonchoSeam: seam, Status: status, Residual: residual}
}

func tool(name, seam string, status PortStatus, residual string) Capability {
	return Capability{Name: name, Kind: "mcp_tool", GonchoSeam: seam, Status: status, Residual: residual}
}

func normalize(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

func toolManifest() []Capability {
	return []Capability{
		tool("memory_recall", "service.Service.Recall", PortDelivered, ""),
		tool("memory_compress_file", "service.ImportFile + summary compaction", PortPartial, "markdown rewrite/backup behavior is not copied"),
		tool("memory_save", "service.Service.Conclude", PortDelivered, ""),
		tool("memory_file_history", "service.Search session/file filters", PortPartial, "dedicated file-history projection remains host-owned"),
		tool("memory_patterns", "service.Search + skill outcome/proposal evidence", PortPartial, "pattern mining is conservative and review-gated"),
		tool("memory_sessions", "session.SessionDirectory", PortAdapter, "host owns authoritative session directory"),
		tool("memory_smart_search", "service.Service.Search", PortDelivered, ""),
		tool("memory_vision_search", "not implemented", PortDeferred, "cross-modal CLIP image embeddings are outside current Go core"),
		tool("memory_timeline", "service timeline annotations + observations", PortDelivered, ""),
		tool("memory_profile", "service.Service.Profile", PortDelivered, ""),
		tool("memory_export", "memory.GonchoMemoryV1Document", PortDelivered, ""),
		tool("memory_relations", "service relation annotations", PortDelivered, ""),
		tool("memory_commit_lookup", "service code-claim verification", PortPartial, "git lookup remains host/live-check owned"),
		tool("memory_commits", "service code-claim verification", PortPartial, "git history enumeration remains host-owned"),
		tool("memory_claude_bridge_sync", "memory.GonchoMemoryV1 markdown import/export", PortAdapter, "Claude-specific bridge files are adapter-owned"),
		tool("memory_graph_query", "service.Service.Recall graph expansion", PortDelivered, ""),
		tool("memory_consolidate", "service dream/session summary lanes", PortPartial, "no autonomous global consolidation worker in core"),
		tool("memory_team_share", "policy ACL + memory scope shared", PortPartial, "server/team sync governance remains adapter-owned"),
		tool("memory_team_feed", "policy ACL + shared scope search", PortPartial, "feed pagination shape is not copied"),
		tool("memory_audit", "service.AuditTrail", PortDelivered, ""),
		tool("memory_governance_delete", "service.Service.Conclude(DeleteID)", PortDelivered, ""),
		tool("memory_snapshot_create", "memory.GonchoMemoryV1Document", PortAdapter, "git snapshot storage is host-owned"),
		tool("memory_action_create", "service review/work-intent queues", PortPartial, "full dependency graph actions are not copied"),
		tool("memory_action_update", "service review/work-intent queues", PortPartial, "full dependency graph actions are not copied"),
		tool("memory_frontier", "service review/work-intent queues", PortPartial, "frontier ranking remains conservative"),
		tool("memory_next", "service review/work-intent queues", PortPartial, "single-next planning remains host-owned"),
		tool("memory_lease", "not implemented", PortDeferred, "exclusive distributed leases require server-mode coordination"),
		tool("memory_routine_run", "skill proposals/outcomes", PortPartial, "routine instantiation is review-gated"),
		tool("memory_signal_send", "not implemented", PortDeferred, "inter-agent messaging requires server/team mode"),
		tool("memory_signal_read", "not implemented", PortDeferred, "inter-agent messaging requires server/team mode"),
		tool("memory_checkpoint", "review queue + verification warnings", PortPartial, "external condition gates remain host-owned"),
		tool("memory_mesh_sync", "not implemented", PortExcluded, "P2P mesh sync is excluded from local embedded Goncho core"),
		tool("memory_sentinel_create", "webhooks + review queue", PortPartial, "event watcher runtime remains host-owned"),
		tool("memory_sentinel_trigger", "webhooks + review queue", PortPartial, "event watcher runtime remains host-owned"),
		tool("memory_sketch_create", "review queue proposals", PortPartial, "ephemeral action graph UI is not copied"),
		tool("memory_sketch_promote", "review queue proposals", PortPartial, "ephemeral action graph UI is not copied"),
		tool("memory_crystallize", "service.Conclude + annotations", PortPartial, "chain compaction avoids provider-mandatory behavior"),
		tool("memory_diagnose", "service diagnostics + queue status", PortDelivered, ""),
		tool("memory_heal", "review queue + diagnostics", PortPartial, "automatic repair is review-gated"),
		tool("memory_facet_tag", "typed memory metadata/tags", PortPartial, "facet grammar is narrower than upstream"),
		tool("memory_facet_query", "service.Search filters", PortPartial, "facet grammar is narrower than upstream"),
		tool("memory_verify", "service.Recall provenance + live-check warnings", PortDelivered, ""),
		tool("memory_lesson_save", "skill learning proposals", PortPartial, "lessons require review before durable trust"),
		tool("memory_lesson_recall", "skill outcomes + service.Search", PortPartial, "lesson projection is host-owned"),
		tool("memory_obsidian_export", "memory.GonchoMemoryV1Document", PortAdapter, "Obsidian-specific filesystem layout is adapter-owned"),
		tool("memory_reflect", "review queue + skill proposals", PortPartial, "LLM reflection is not mandatory in core"),
		tool("memory_insight_list", "service.Search + review list", PortPartial, "insight list projection is not copied"),
		tool("memory_slot_list", "not implemented", PortDeferred, "slot memory API is not part of current Goncho contract"),
		tool("memory_slot_get", "not implemented", PortDeferred, "slot memory API is not part of current Goncho contract"),
		tool("memory_slot_create", "not implemented", PortDeferred, "slot memory API is not part of current Goncho contract"),
		tool("memory_slot_append", "not implemented", PortDeferred, "slot memory API is not part of current Goncho contract"),
		tool("memory_slot_replace", "not implemented", PortDeferred, "slot memory API is not part of current Goncho contract"),
		tool("memory_slot_delete", "not implemented", PortDeferred, "slot memory API is not part of current Goncho contract"),
	}
}

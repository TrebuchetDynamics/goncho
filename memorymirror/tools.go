package memorymirror

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	goncho "github.com/TrebuchetDynamics/goncho/service"
	"github.com/TrebuchetDynamics/goncho/toolmeta"
)

type ToolRegistryOptions struct {
	DefaultWorkspaceID string
	DefaultProfileID   string
	DefaultPeerID      string
	DefaultSessionKey  string
}

func NewToolRegistry(svc *goncho.Service, opts ToolRegistryOptions) []toolmeta.Tool {
	return []toolmeta.Tool{
		newServiceTool("memory_save", "Explicitly save an important insight, decision, or pattern to Goncho long-term memory using broad-memory-compatible arguments.", json.RawMessage(`{"type":"object","properties":{"content":{"type":"string","description":"The insight or decision to remember"},"type":{"type":"string","description":"Memory type: pattern, preference, architecture, bug, workflow, or fact"},"concepts":{"type":"string","description":"Comma-separated key concepts"},"files":{"type":"string","description":"Comma-separated relevant file paths"},"peer_id":{"type":"string","description":"Optional Goncho peer id"},"session_id":{"type":"string","description":"Optional session id"}},"required":["content"]}`), true, false, func(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
			return executeMemorySave(ctx, svc, opts, args)
		}),
		newServiceTool("memory_smart_search", "Hybrid Goncho search exposed through the compatible memory_smart_search name.", json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","description":"Search query"},"expandIds":{"type":"string","description":"Comma-separated observation IDs to expand"},"limit":{"type":"number","description":"Max results (default 10)"},"peer_id":{"type":"string","description":"Optional Goncho peer id"},"session_id":{"type":"string","description":"Optional session id"}},"required":["query"]}`), false, true, func(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
			return executeMemorySmartSearch(ctx, svc, opts, args)
		}),
		newServiceTool("memory_recall", "Search past Goncho memory for relevant context using the compatible memory_recall name.", json.RawMessage(`{"type":"object","properties":{"query":{"type":"string","description":"Search query"},"limit":{"type":"number","description":"Max results to return (default 10)"},"format":{"type":"string","description":"Result format: full, compact, or narrative (default full)"},"token_budget":{"type":"number","description":"Optional token budget to trim returned results"},"peer_id":{"type":"string","description":"Optional Goncho peer id"},"session_id":{"type":"string","description":"Optional session id"}},"required":["query"]}`), false, true, func(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
			return executeMemoryRecall(ctx, svc, opts, args)
		}),
		newServiceTool("memory_profile", "Return the Goncho peer profile through the compatible memory_profile name.", json.RawMessage(`{"type":"object","properties":{"peer_id":{"type":"string","description":"Optional Goncho peer id"}}}`), false, true, func(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
			return executeMemoryProfile(ctx, svc, opts, args)
		}),
	}
}

type serviceTool struct {
	name        string
	description string
	schema      json.RawMessage
	mutating    bool
	idempotent  bool
	execute     func(context.Context, json.RawMessage) (json.RawMessage, error)
}

func newServiceTool(name, description string, schema json.RawMessage, mutating, idempotent bool, execute func(context.Context, json.RawMessage) (json.RawMessage, error)) *serviceTool {
	return &serviceTool{name: name, description: description, schema: schema, mutating: mutating, idempotent: idempotent, execute: execute}
}

func (t *serviceTool) Name() string { return t.name }

func (t *serviceTool) Description() string { return t.description }

func (t *serviceTool) Schema() json.RawMessage { return append(json.RawMessage(nil), t.schema...) }

func (t *serviceTool) Timeout() time.Duration { return 5 * time.Second }

func (t *serviceTool) Spec() toolmeta.OperationSpec {
	return toolmeta.OperationSpec{ToolDescriptor: toolmeta.ToolDescriptor{Name: t.name, Description: t.description, Schema: t.Schema()}, Mutating: t.mutating, Idempotent: t.idempotent, PromptSafe: true, TrustClass: []string{"operator", "child-agent", "system"}, AuditKind: "memory"}
}

func (t *serviceTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	if t == nil || t.execute == nil {
		return nil, errors.New("memorymirror: tool executor is required")
	}
	return t.execute(ctx, args)
}

func executeMemorySave(ctx context.Context, svc *goncho.Service, opts ToolRegistryOptions, args json.RawMessage) (json.RawMessage, error) {
	if svc == nil {
		return nil, errors.New("memory_save: service is required")
	}
	var in struct {
		Content   string `json:"content"`
		Type      string `json:"type"`
		Concepts  string `json:"concepts"`
		Files     string `json:"files"`
		PeerID    string `json:"peer_id"`
		Peer      string `json:"peer"`
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("memory_save: %w", err)
	}
	content := strings.TrimSpace(in.Content)
	if content == "" {
		return nil, errors.New("memory_save: content is required")
	}
	if extra := upstreamMetadataSuffix(in.Type, in.Concepts, in.Files); extra != "" {
		content += "\n" + extra
	}
	out, err := svc.Conclude(ctx, goncho.ConcludeParams{ProfileID: opts.DefaultProfileID, Peer: peerFromInputs(opts, in.PeerID, in.Peer), Conclusion: content, SessionKey: firstNonEmpty(in.SessionID, opts.DefaultSessionKey), Scope: strings.TrimSpace(in.Type)})
	if err != nil {
		return nil, fmt.Errorf("memory_save: %w", err)
	}
	return json.Marshal(map[string]any{"success": true, "tool": "memory_save", "backend": "goncho", "retrieval": "goncho_conclude", "id": out.ID, "status": out.Status, "peer": out.Peer, "workspace_id": out.WorkspaceID, "profile_id": out.ProfileID, "local_first": true})
}

func executeMemorySmartSearch(ctx context.Context, svc *goncho.Service, opts ToolRegistryOptions, args json.RawMessage) (json.RawMessage, error) {
	if svc == nil {
		return nil, errors.New("memory_smart_search: service is required")
	}
	var in struct {
		Query     string `json:"query"`
		Limit     int    `json:"limit"`
		PeerID    string `json:"peer_id"`
		Peer      string `json:"peer"`
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("memory_smart_search: %w", err)
	}
	if strings.TrimSpace(in.Query) == "" {
		return nil, errors.New("memory_smart_search: query is required")
	}
	out, err := svc.Search(ctx, goncho.SearchParams{ProfileID: opts.DefaultProfileID, Peer: peerFromInputs(opts, in.PeerID, in.Peer), Query: in.Query, SessionKey: firstNonEmpty(in.SessionID, opts.DefaultSessionKey)})
	if err != nil {
		return nil, fmt.Errorf("memory_smart_search: %w", err)
	}
	results := out.Results
	if in.Limit > 0 && len(results) > in.Limit {
		results = results[:in.Limit]
	}
	return json.Marshal(map[string]any{"success": true, "tool": "memory_smart_search", "backend": "goncho", "retrieval": "goncho_search", "query": out.Query, "count": len(results), "results": results, "workspace_id": out.WorkspaceID, "profile_id": out.ProfileID, "peer": out.Peer, "local_first": true})
}

func executeMemoryRecall(ctx context.Context, svc *goncho.Service, opts ToolRegistryOptions, args json.RawMessage) (json.RawMessage, error) {
	if svc == nil {
		return nil, errors.New("memory_recall: service is required")
	}
	var in struct {
		Query       string `json:"query"`
		Limit       int    `json:"limit"`
		Format      string `json:"format"`
		TokenBudget int    `json:"token_budget"`
		PeerID      string `json:"peer_id"`
		Peer        string `json:"peer"`
		SessionID   string `json:"session_id"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("memory_recall: %w", err)
	}
	if strings.TrimSpace(in.Query) == "" {
		return nil, errors.New("memory_recall: query is required")
	}
	trace, err := svc.Recall(ctx, goncho.RecallQuery{WorkspaceID: opts.DefaultWorkspaceID, Peer: peerFromInputs(opts, in.PeerID, in.Peer), Query: in.Query, SessionKey: firstNonEmpty(in.SessionID, opts.DefaultSessionKey), Limit: in.Limit, MaxTokens: in.TokenBudget})
	if err != nil {
		return nil, fmt.Errorf("memory_recall: %w", err)
	}
	selected := trace.Selected
	return json.Marshal(map[string]any{"success": true, "tool": "memory_recall", "backend": "goncho", "retrieval": "goncho_recall", "format": firstNonEmpty(in.Format, "full"), "trace_id": trace.TraceID, "selected_count": len(selected), "results": selected, "warnings": trace.Warnings, "local_first": true})
}

func executeMemoryProfile(ctx context.Context, svc *goncho.Service, opts ToolRegistryOptions, args json.RawMessage) (json.RawMessage, error) {
	if svc == nil {
		return nil, errors.New("memory_profile: service is required")
	}
	var in struct {
		PeerID string `json:"peer_id"`
		Peer   string `json:"peer"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("memory_profile: %w", err)
	}
	out, err := svc.Profile(ctx, peerFromInputs(opts, in.PeerID, in.Peer))
	if err != nil {
		return nil, fmt.Errorf("memory_profile: %w", err)
	}
	return json.Marshal(map[string]any{"success": true, "tool": "memory_profile", "backend": "goncho", "profile": out, "local_first": true})
}

func upstreamMetadataSuffix(memoryType, concepts, files string) string {
	var parts []string
	if strings.TrimSpace(memoryType) != "" {
		parts = append(parts, "upstream_memory.type="+strings.TrimSpace(memoryType))
	}
	if strings.TrimSpace(concepts) != "" {
		parts = append(parts, "upstream_memory.concepts="+strings.TrimSpace(concepts))
	}
	if strings.TrimSpace(files) != "" {
		parts = append(parts, "upstream_memory.files="+strings.TrimSpace(files))
	}
	if len(parts) == 0 {
		return ""
	}
	return "[" + strings.Join(parts, "; ") + "]"
}

func peerFromInputs(opts ToolRegistryOptions, values ...string) string {
	return firstNonEmpty(append(values, opts.DefaultPeerID, "memorymirror")...)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

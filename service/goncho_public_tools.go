package goncho

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	toolmeta "github.com/TrebuchetDynamics/goncho/toolmeta"
)

type GonchoContextTool struct{ svc *Service }

type GonchoSearchTool struct{ svc *Service }

type GonchoRecallTool struct{ svc *Service }

type GonchoRememberTool struct{ svc *Service }

type GonchoHandoffTool struct{ store MemoryToolStore }

func NewGonchoContextTool(svc *Service) *GonchoContextTool { return &GonchoContextTool{svc: svc} }
func NewGonchoSearchTool(svc *Service) *GonchoSearchTool   { return &GonchoSearchTool{svc: svc} }
func NewGonchoRecallTool(svc *Service) *GonchoRecallTool   { return &GonchoRecallTool{svc: svc} }
func NewGonchoRememberTool(svc *Service) *GonchoRememberTool {
	return &GonchoRememberTool{svc: svc}
}
func NewGonchoHandoffTool(store MemoryToolStore) *GonchoHandoffTool {
	return &GonchoHandoffTool{store: store}
}

func (t *GonchoContextTool) Name() string           { return "goncho_context" }
func (t *GonchoContextTool) Timeout() time.Duration { return 5 * time.Second }
func (t *GonchoContextTool) Description() string {
	return "Build a local Goncho orientation pack for a peer and session."
}
func (t *GonchoContextTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"profile_id":{"type":"string"},"peer_id":{"type":"string"},"query":{"type":"string"},"session_key":{"type":"string"},"max_tokens":{"type":"integer"}},"required":["peer_id"]}`)
}
func (t *GonchoContextTool) Spec() toolmeta.OperationSpec {
	return gonchoPublicToolSpec(t.Name(), t.Description(), t.Schema(), false, true)
}
func (t *GonchoContextTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	if t == nil || t.svc == nil {
		return nil, errors.New("goncho_context: service is required")
	}
	var in struct {
		ProfileID  string `json:"profile_id"`
		PeerID     string `json:"peer_id"`
		Peer       string `json:"peer"`
		Query      string `json:"query"`
		SessionKey string `json:"session_key"`
		MaxTokens  int    `json:"max_tokens"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("goncho_context: %w", err)
	}
	out, err := t.svc.Context(ctx, ContextParams{ProfileID: in.ProfileID, Peer: firstPublicNonEmpty(in.PeerID, in.Peer), Query: in.Query, SessionKey: in.SessionKey, MaxTokens: in.MaxTokens})
	if err != nil {
		return nil, fmt.Errorf("goncho_context: %w", err)
	}
	return json.Marshal(out)
}

func (t *GonchoSearchTool) Name() string           { return "goncho_search" }
func (t *GonchoSearchTool) Timeout() time.Duration { return 5 * time.Second }
func (t *GonchoSearchTool) Description() string {
	return "Search local Goncho memory with peer, session, scope, and token controls."
}
func (t *GonchoSearchTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"profile_id":{"type":"string"},"peer_id":{"type":"string"},"query":{"type":"string"},"session_key":{"type":"string"},"scope":{"type":"string"},"max_tokens":{"type":"integer"}},"required":["peer_id","query"]}`)
}
func (t *GonchoSearchTool) Spec() toolmeta.OperationSpec {
	return gonchoPublicToolSpec(t.Name(), t.Description(), t.Schema(), false, true)
}
func (t *GonchoSearchTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	if t == nil || t.svc == nil {
		return nil, errors.New("goncho_search: service is required")
	}
	var in struct {
		ProfileID  string `json:"profile_id"`
		PeerID     string `json:"peer_id"`
		Peer       string `json:"peer"`
		Query      string `json:"query"`
		SessionKey string `json:"session_key"`
		Scope      string `json:"scope"`
		MaxTokens  int    `json:"max_tokens"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("goncho_search: %w", err)
	}
	out, err := t.svc.Search(ctx, SearchParams{ProfileID: in.ProfileID, Peer: firstPublicNonEmpty(in.PeerID, in.Peer), Query: in.Query, SessionKey: in.SessionKey, Scope: in.Scope, MaxTokens: in.MaxTokens})
	if err != nil {
		return nil, fmt.Errorf("goncho_search: %w", err)
	}
	return json.Marshal(map[string]any{"action": "search", "count": len(out.Results), "results": out.Results, "workspace_id": out.WorkspaceID, "profile_id": out.ProfileID, "peer": out.Peer, "query": out.Query})
}

func (t *GonchoRecallTool) Name() string           { return "goncho_recall" }
func (t *GonchoRecallTool) Timeout() time.Duration { return 5 * time.Second }
func (t *GonchoRecallTool) Description() string {
	return "Run auditable Goncho recall and return the scored trace with replay evidence."
}
func (t *GonchoRecallTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"workspace_id":{"type":"string"},"peer_id":{"type":"string"},"peer":{"type":"string"},"query":{"type":"string"},"session_key":{"type":"string"},"scope":{"type":"string"},"scope_id":{"type":"string"},"sources":{"type":"array","items":{"type":"string"}},"limit":{"type":"integer"},"max_tokens":{"type":"integer"},"compact":{"type":"boolean"}},"required":["peer_id","query"]}`)
}
func (t *GonchoRecallTool) Spec() toolmeta.OperationSpec {
	return gonchoPublicToolSpec(t.Name(), t.Description(), t.Schema(), false, true)
}
func (t *GonchoRecallTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	if t == nil || t.svc == nil {
		return nil, errors.New("goncho_recall: service is required")
	}
	var in struct {
		WorkspaceID string   `json:"workspace_id"`
		PeerID      string   `json:"peer_id"`
		Peer        string   `json:"peer"`
		Query       string   `json:"query"`
		SessionKey  string   `json:"session_key"`
		Scope       string   `json:"scope"`
		ScopeID     string   `json:"scope_id"`
		Sources     []string `json:"sources"`
		Limit       int      `json:"limit"`
		MaxTokens   int      `json:"max_tokens"`
		Compact     bool     `json:"compact"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("goncho_recall: %w", err)
	}
	trace, err := t.svc.Recall(ctx, RecallQuery{
		WorkspaceID: in.WorkspaceID,
		Peer:        firstPublicNonEmpty(in.PeerID, in.Peer),
		Query:       in.Query,
		SessionKey:  in.SessionKey,
		ScopeID:     firstPublicNonEmpty(in.ScopeID, in.Scope),
		Sources:     in.Sources,
		Limit:       in.Limit,
		MaxTokens:   in.MaxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("goncho_recall: %w", err)
	}
	return json.Marshal(buildGonchoRecallToolOutput(trace, in.Compact))
}

func (t *GonchoRememberTool) Name() string           { return "goncho_remember" }
func (t *GonchoRememberTool) Timeout() time.Duration { return 5 * time.Second }
func (t *GonchoRememberTool) Description() string {
	return "Store an explicit local Goncho claim for a peer."
}
func (t *GonchoRememberTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"profile_id":{"type":"string"},"peer_id":{"type":"string"},"content":{"type":"string"},"session_key":{"type":"string"},"scope":{"type":"string"}},"required":["peer_id","content"]}`)
}
func (t *GonchoRememberTool) Spec() toolmeta.OperationSpec {
	return gonchoPublicToolSpec(t.Name(), t.Description(), t.Schema(), true, false)
}
func (t *GonchoRememberTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	if t == nil || t.svc == nil {
		return nil, errors.New("goncho_remember: service is required")
	}
	var in struct {
		ProfileID  string `json:"profile_id"`
		PeerID     string `json:"peer_id"`
		Peer       string `json:"peer"`
		Content    string `json:"content"`
		Conclusion string `json:"conclusion"`
		SessionKey string `json:"session_key"`
		Scope      string `json:"scope"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("goncho_remember: %w", err)
	}
	out, err := t.svc.Conclude(ctx, ConcludeParams{ProfileID: in.ProfileID, Peer: firstPublicNonEmpty(in.PeerID, in.Peer), Conclusion: firstPublicNonEmpty(in.Content, in.Conclusion), SessionKey: in.SessionKey, Scope: in.Scope})
	if err != nil {
		return nil, fmt.Errorf("goncho_remember: %w", err)
	}
	return json.Marshal(map[string]any{"success": true, "action": "remember", "id": out.ID, "status": out.Status, "peer": out.Peer, "workspace_id": out.WorkspaceID, "profile_id": out.ProfileID})
}

func (t *GonchoHandoffTool) Name() string           { return "goncho_handoff" }
func (t *GonchoHandoffTool) Timeout() time.Duration { return 5 * time.Second }
func (t *GonchoHandoffTool) Description() string {
	return "Save or load local handoff/prospective memory for a session."
}
func (t *GonchoHandoffTool) Schema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"action":{"type":"string","enum":["save","load"]},"session_id":{"type":"string"},"content":{"type":"string"},"limit":{"type":"integer"}},"required":["action","session_id"]}`)
}
func (t *GonchoHandoffTool) Spec() toolmeta.OperationSpec {
	return gonchoPublicToolSpec(t.Name(), t.Description(), t.Schema(), true, false)
}
func (t *GonchoHandoffTool) Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error) {
	if t == nil || t.store == nil {
		return nil, errors.New("goncho_handoff: store is required")
	}
	var in struct {
		Action    string `json:"action"`
		SessionID string `json:"session_id"`
		Content   string `json:"content"`
		Limit     int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("goncho_handoff: %w", err)
	}
	switch strings.TrimSpace(in.Action) {
	case "save":
		if strings.TrimSpace(in.SessionID) == "" || strings.TrimSpace(in.Content) == "" {
			return nil, errors.New("goncho_handoff: save requires session_id and content")
		}
		now := time.Now().UTC()
		entry := MemoryToolEntry{ID: fmt.Sprintf("handoff_%d", now.UnixNano()), Content: in.Content, Tags: []string{"handoff", in.SessionID}, Importance: 0.8, SessionID: in.SessionID, CreatedAt: now, UpdatedAt: now, Metadata: map[string]string{"type": "handoff"}}
		if err := t.store.Store(ctx, entry); err != nil {
			return nil, fmt.Errorf("goncho_handoff save: %w", err)
		}
		return json.Marshal(map[string]any{"success": true, "action": "save", "id": entry.ID, "session_id": entry.SessionID})
	case "load":
		limit := in.Limit
		if limit <= 0 {
			limit = 10
		}
		entries, err := t.store.Retrieve(ctx, "handoff", limit)
		if err != nil {
			return nil, fmt.Errorf("goncho_handoff load: %w", err)
		}
		filtered := make([]MemoryToolEntry, 0, len(entries))
		for _, entry := range entries {
			if entry.SessionID == in.SessionID {
				filtered = append(filtered, entry)
			}
		}
		return json.Marshal(map[string]any{"success": true, "action": "load", "count": len(filtered), "items": filtered})
	default:
		return nil, errors.New("goncho_handoff: action must be save or load")
	}
}

func gonchoPublicToolSpec(name, description string, schema json.RawMessage, mutating, idempotent bool) toolmeta.OperationSpec {
	return toolmeta.OperationSpec{ToolDescriptor: toolmeta.ToolDescriptor{Name: name, Description: description, Schema: schema}, Mutating: mutating, Idempotent: idempotent, PromptSafe: true, TrustClass: []string{"operator", "system"}, AuditKind: "memory"}
}

func firstPublicNonEmpty(values ...string) string {
	return firstNonBlank(values...)
}

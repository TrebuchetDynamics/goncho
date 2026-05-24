package goncho

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/goncho/memory"
)

type GonchoPublicToolsRestartE2EConfig struct {
	DBPath       string
	MarkdownPath string
	WorkspaceID  string
	ObserverID   string
	PeerID       string
	SessionKey   string
}

type GonchoPublicToolsRestartE2EReport struct {
	ToolNames                         []string `json:"tool_names"`
	SQLiteRestartVerified             bool     `json:"sqlite_restart_verified"`
	NetworkRequired                   bool     `json:"network_required"`
	OllamaRequired                    bool     `json:"ollama_required"`
	SearchCountBeforeRestart          int      `json:"search_count_before_restart"`
	SearchCountAfterRestart           int      `json:"search_count_after_restart"`
	RecallSelectedAfterRestart        int      `json:"recall_selected_after_restart"`
	ContextRepresentationAfterRestart string   `json:"context_representation_after_restart"`
	ReviewWarningBeforeResolve        bool     `json:"review_warning_before_resolve"`
	ReviewWarningAfterResolve         bool     `json:"review_warning_after_resolve"`
	HandoffCountAfterRestart          int      `json:"handoff_count_after_restart"`
	CompletionCondition               string   `json:"completion_condition"`
}

func RunGonchoPublicToolsRestartE2E(ctx context.Context, cfg GonchoPublicToolsRestartE2EConfig) (GonchoPublicToolsRestartE2EReport, error) {
	cfg = cfg.withDefaults()
	store, err := memory.OpenSqlite(cfg.DBPath, 0, nil)
	if err != nil {
		return GonchoPublicToolsRestartE2EReport{}, err
	}
	if err := RunMigrations(store.DB()); err != nil {
		_ = store.Close(ctx)
		return GonchoPublicToolsRestartE2EReport{}, err
	}
	svc := NewService(store.DB(), Config{WorkspaceID: cfg.WorkspaceID, ObserverPeerID: cfg.ObserverID, RecentMessages: 4}, nil)
	memoryStore := NewLocalMarkdownMemoryStore(store.DB(), LocalMarkdownMemoryConfig{Path: cfg.MarkdownPath, AgentID: cfg.ObserverID, WorkspaceID: cfg.WorkspaceID, ObserverPeerID: cfg.ObserverID, PeerID: cfg.PeerID, SessionID: cfg.SessionKey})
	rememberTool := NewGonchoRememberTool(svc)
	searchTool := NewGonchoSearchTool(svc)
	recallTool := NewGonchoRecallTool(svc)
	contextTool := NewGonchoContextTool(svc)
	reviewTool := NewReviewTool(svc)
	handoffTool := NewGonchoHandoffTool(memoryStore)

	if _, err := executePublicToolMap(ctx, rememberTool, map[string]any{"peer_id": cfg.PeerID, "content": "Goncho public tools restart E2E persists local-first claims.", "session_key": cfg.SessionKey}); err != nil {
		_ = store.Close(ctx)
		return GonchoPublicToolsRestartE2EReport{}, err
	}
	beforeSearch, err := executePublicToolMap(ctx, searchTool, map[string]any{"peer_id": cfg.PeerID, "query": "restart E2E", "session_key": cfg.SessionKey})
	if err != nil {
		_ = store.Close(ctx)
		return GonchoPublicToolsRestartE2EReport{}, err
	}

	item, err := svc.CreateReviewItem(ctx, ReviewItemCreateParams{Kind: ReviewKindStale, PeerID: cfg.PeerID, SessionKey: cfg.SessionKey, SubjectID: "restart-memory", Reason: "restart E2E review warning"})
	if err != nil {
		_ = store.Close(ctx)
		return GonchoPublicToolsRestartE2EReport{}, err
	}
	beforeContext, err := svc.Context(ctx, ContextParams{Peer: cfg.PeerID, SessionKey: cfg.SessionKey})
	if err != nil {
		_ = store.Close(ctx)
		return GonchoPublicToolsRestartE2EReport{}, err
	}
	if _, err := executePublicToolMap(ctx, reviewTool, map[string]any{"action": "resolve", "id": item.ID, "resolution": "verified", "resolved_by": cfg.ObserverID, "resolution_reason": "restart E2E checked"}); err != nil {
		_ = store.Close(ctx)
		return GonchoPublicToolsRestartE2EReport{}, err
	}
	afterContext, err := svc.Context(ctx, ContextParams{Peer: cfg.PeerID, SessionKey: cfg.SessionKey})
	if err != nil {
		_ = store.Close(ctx)
		return GonchoPublicToolsRestartE2EReport{}, err
	}
	if _, err := executePublicToolMap(ctx, handoffTool, map[string]any{"action": "save", "session_id": cfg.SessionKey, "content": "After restart, run go test ./... before claiming Goncho done."}); err != nil {
		_ = store.Close(ctx)
		return GonchoPublicToolsRestartE2EReport{}, err
	}
	if err := store.Close(ctx); err != nil {
		return GonchoPublicToolsRestartE2EReport{}, err
	}

	reopened, err := memory.OpenSqlite(cfg.DBPath, 0, nil)
	if err != nil {
		return GonchoPublicToolsRestartE2EReport{}, err
	}
	defer reopened.Close(ctx)
	if err := RunMigrations(reopened.DB()); err != nil {
		return GonchoPublicToolsRestartE2EReport{}, err
	}
	reopenedSvc := NewService(reopened.DB(), Config{WorkspaceID: cfg.WorkspaceID, ObserverPeerID: cfg.ObserverID, RecentMessages: 4}, nil)
	reopenedMemoryStore := NewLocalMarkdownMemoryStore(reopened.DB(), LocalMarkdownMemoryConfig{Path: cfg.MarkdownPath, AgentID: cfg.ObserverID, WorkspaceID: cfg.WorkspaceID, ObserverPeerID: cfg.ObserverID, PeerID: cfg.PeerID, SessionID: cfg.SessionKey})
	reopenedSearchTool := NewGonchoSearchTool(reopenedSvc)
	reopenedRecallTool := NewGonchoRecallTool(reopenedSvc)
	reopenedContextTool := NewGonchoContextTool(reopenedSvc)
	reopenedHandoffTool := NewGonchoHandoffTool(reopenedMemoryStore)

	afterSearch, err := executePublicToolMap(ctx, reopenedSearchTool, map[string]any{"peer_id": cfg.PeerID, "query": "restart E2E", "session_key": cfg.SessionKey})
	if err != nil {
		return GonchoPublicToolsRestartE2EReport{}, err
	}
	afterRecall, err := executePublicToolMap(ctx, reopenedRecallTool, map[string]any{"peer_id": cfg.PeerID, "query": "restart E2E", "session_key": cfg.SessionKey, "limit": 3})
	if err != nil {
		return GonchoPublicToolsRestartE2EReport{}, err
	}
	afterContextMap, err := executePublicToolMap(ctx, reopenedContextTool, map[string]any{"peer_id": cfg.PeerID, "query": "restart E2E", "session_key": cfg.SessionKey})
	if err != nil {
		return GonchoPublicToolsRestartE2EReport{}, err
	}
	handoff, err := executePublicToolMap(ctx, reopenedHandoffTool, map[string]any{"action": "load", "session_id": cfg.SessionKey})
	if err != nil {
		return GonchoPublicToolsRestartE2EReport{}, err
	}

	return GonchoPublicToolsRestartE2EReport{
		ToolNames:                         []string{rememberTool.Name(), searchTool.Name(), recallTool.Name(), contextTool.Name(), reviewTool.Name(), handoffTool.Name()},
		SQLiteRestartVerified:             publicToolInt(afterSearch, "count") == 1 && publicToolInt(afterRecall, "selected_count") == 1 && strings.TrimSpace(publicToolString(afterContextMap, "representation")) != "",
		NetworkRequired:                   false,
		OllamaRequired:                    false,
		SearchCountBeforeRestart:          publicToolInt(beforeSearch, "count"),
		SearchCountAfterRestart:           publicToolInt(afterSearch, "count"),
		RecallSelectedAfterRestart:        publicToolInt(afterRecall, "selected_count"),
		ContextRepresentationAfterRestart: publicToolString(afterContextMap, "representation"),
		ReviewWarningBeforeResolve:        contextUnavailableHasPublicCapability(beforeContext.Unavailable, "review_required"),
		ReviewWarningAfterResolve:         contextUnavailableHasPublicCapability(afterContext.Unavailable, "review_required"),
		HandoffCountAfterRestart:          publicToolInt(handoff, "count"),
		CompletionCondition:               "go test ./...",
	}, nil
}

func (cfg GonchoPublicToolsRestartE2EConfig) withDefaults() GonchoPublicToolsRestartE2EConfig {
	if cfg.WorkspaceID == "" {
		cfg.WorkspaceID = "restart-e2e-workspace"
	}
	if cfg.ObserverID == "" {
		cfg.ObserverID = "agent:mineru"
	}
	if cfg.PeerID == "" {
		cfg.PeerID = "peer-restart-e2e"
	}
	if cfg.SessionKey == "" {
		cfg.SessionKey = "session-restart-e2e"
	}
	return cfg
}

func executePublicToolMap(ctx context.Context, tool interface {
	Execute(context.Context, json.RawMessage) (json.RawMessage, error)
}, args map[string]any) (map[string]any, error) {
	rawArgs, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	rawOut, err := tool.Execute(ctx, rawArgs)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(rawOut, &out); err != nil {
		return nil, fmt.Errorf("decode public tool output: %w", err)
	}
	return out, nil
}

func publicToolInt(values map[string]any, key string) int {
	switch value := values[key].(type) {
	case int:
		return value
	case float64:
		return int(value)
	default:
		return 0
	}
}

func publicToolString(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return value
}

func contextUnavailableHasPublicCapability(values []ContextUnavailableEvidence, capability string) bool {
	for _, value := range values {
		if value.Capability == capability {
			return true
		}
	}
	return false
}

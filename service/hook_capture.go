package goncho

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// HostHookEventName is a host-neutral lifecycle/tool event name accepted by
// CaptureHostHook. Host adapters translate Claude Code, Pi, MCP, or other
// runtime-specific hooks into this shape before handing them to Goncho.
type HostHookEventName string

const (
	HostHookSessionStart      HostHookEventName = "session_start"
	HostHookUserPrompt        HostHookEventName = "user_prompt"
	HostHookPostToolUse       HostHookEventName = "post_tool_use"
	HostHookAssistantResponse HostHookEventName = "assistant_response"
	HostHookCompact           HostHookEventName = "compact"
	HostHookSessionEnd        HostHookEventName = "session_end"
	HostHookFailure           HostHookEventName = "failure"
)

// HostHookEvent is the host-neutral automatic capture envelope. It is designed
// to be small enough for hook scripts and rich enough to route to Observe,
// CreateMessages, and session-summary persistence without importing a host SDK.
type HostHookEvent struct {
	Event       HostHookEventName `json:"event"`
	Host        string            `json:"host,omitempty"`
	WorkspaceID string            `json:"workspace_id,omitempty"`
	ProfileID   string            `json:"profile_id,omitempty"`
	PeerID      string            `json:"peer_id,omitempty"`
	SessionKey  string            `json:"session_key,omitempty"`
	ContextID   string            `json:"context_id,omitempty"`
	ToolName    string            `json:"tool_name,omitempty"`
	Content     string            `json:"content,omitempty"`
	Input       string            `json:"input,omitempty"`
	Output      string            `json:"output,omitempty"`
	Error       string            `json:"error,omitempty"`
	Summary     string            `json:"summary,omitempty"`
	Success     *bool             `json:"success,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	ObservedAt  time.Time         `json:"observed_at,omitempty"`
}

// HookCaptureResult reports every durable write performed for a host hook.
type HookCaptureResult struct {
	Observations []Observation   `json:"observations"`
	Messages     []MessageRecord `json:"messages"`
	Summary      *SessionSummary `json:"summary,omitempty"`
}

// CaptureHostHook records a host-neutral hook event into Goncho's observation
// log and, for conversational events, the session message stream.
func (s *Service) CaptureHostHook(ctx context.Context, event HostHookEvent) (HookCaptureResult, error) {
	if s == nil {
		return HookCaptureResult{}, fmt.Errorf("goncho: nil service")
	}
	kind, err := observationKindForHostHook(event)
	if err != nil {
		return HookCaptureResult{}, err
	}
	metadata := hostHookMetadata(event)
	obsInput, obsOutput := hostHookObservationIO(event)
	result := HookCaptureResult{Observations: []Observation{}, Messages: []MessageRecord{}}
	observed, err := s.Observe(ctx, ObservationParams{
		Kind:        kind,
		WorkspaceID: event.WorkspaceID,
		ProfileID:   event.ProfileID,
		PeerID:      event.PeerID,
		SessionKey:  event.SessionKey,
		ContextID:   event.ContextID,
		Input:       obsInput,
		Output:      obsOutput,
		Success:     hostHookSuccess(event),
		Metadata:    metadata,
		ObservedAt:  event.ObservedAt,
		Reason:      "host_hook_capture",
	})
	if err != nil {
		return HookCaptureResult{}, err
	}
	result.Observations = append(result.Observations, observed.Observation)

	if role := hostHookMessageRole(event.Event); role != "" {
		content := strings.TrimSpace(firstNonBlank(event.Content, event.Input, event.Output))
		if content == "" {
			return HookCaptureResult{}, fmt.Errorf("goncho: hook %s requires message content", event.Event)
		}
		created, err := s.CreateMessages(ctx, CreateMessagesParams{
			SessionKey: strings.TrimSpace(event.SessionKey),
			Messages: []CreateMessage{{
				ProfileID: event.ProfileID,
				Peer:      event.PeerID,
				Role:      role,
				Content:   content,
				Metadata:  stringMapToAny(metadata),
				CreatedAt: event.ObservedAt,
			}},
		})
		if err != nil {
			return HookCaptureResult{}, err
		}
		result.Messages = append(result.Messages, created.Messages...)
	}

	if event.Event == HostHookSessionEnd && strings.TrimSpace(event.Summary) != "" {
		summary := SessionSummary{
			Content:     strings.TrimSpace(event.Summary),
			SummaryType: "short",
			CreatedAt:   hostHookUnix(event.ObservedAt),
			TokenCount:  approxTokens(event.Summary),
		}
		if err := upsertSessionSummary(ctx, s.db, sessionSummaryRow{
			WorkspaceID: serviceObservationWorkspace(s.workspaceID, event.WorkspaceID),
			SessionKey:  strings.TrimSpace(event.SessionKey),
			SummaryType: summary.SummaryType,
			Content:     summary.Content,
			CreatedAt:   summary.CreatedAt,
			TokenCount:  summary.TokenCount,
		}); err != nil {
			return HookCaptureResult{}, err
		}
		result.Summary = &summary
	}
	return result, nil
}

func observationKindForHostHook(event HostHookEvent) (ObservationKind, error) {
	switch event.Event {
	case HostHookSessionStart:
		return ObservationKindSessionStart, nil
	case HostHookUserPrompt:
		return ObservationKindUserPrompt, nil
	case HostHookPostToolUse:
		if hostHookFailed(event) {
			return ObservationKindToolError, nil
		}
		return ObservationKindToolResult, nil
	case HostHookAssistantResponse:
		return ObservationKindAssistantResponse, nil
	case HostHookCompact:
		return ObservationKindCompact, nil
	case HostHookSessionEnd:
		return ObservationKindSessionEnd, nil
	case HostHookFailure:
		return ObservationKindToolError, nil
	default:
		return "", fmt.Errorf("goncho: unsupported host hook event %q", event.Event)
	}
}

func hostHookObservationIO(event HostHookEvent) (string, string) {
	switch event.Event {
	case HostHookUserPrompt:
		return firstNonBlank(event.Content, event.Input), ""
	case HostHookAssistantResponse:
		return event.Input, firstNonBlank(event.Content, event.Output)
	case HostHookSessionEnd:
		return event.Input, firstNonBlank(event.Summary, event.Output, event.Content)
	default:
		return firstNonBlank(event.Input, event.Content), firstNonBlank(event.Output, event.Error, event.Summary)
	}
}

func hostHookMessageRole(event HostHookEventName) string {
	switch event {
	case HostHookUserPrompt:
		return "user"
	case HostHookAssistantResponse:
		return "assistant"
	default:
		return ""
	}
}

func hostHookSuccess(event HostHookEvent) *bool {
	if event.Success != nil {
		return event.Success
	}
	switch event.Event {
	case HostHookPostToolUse, HostHookFailure:
		value := !hostHookFailed(event)
		return &value
	default:
		return nil
	}
}

func hostHookFailed(event HostHookEvent) bool {
	return strings.TrimSpace(event.Error) != "" || (event.Success != nil && !*event.Success) || event.Event == HostHookFailure
}

func hostHookMetadata(event HostHookEvent) map[string]string {
	out := make(map[string]string, len(event.Metadata)+3)
	for key, value := range event.Metadata {
		out[key] = value
	}
	out["hook_event"] = string(event.Event)
	if strings.TrimSpace(event.Host) != "" {
		out["host"] = strings.TrimSpace(event.Host)
	}
	if strings.TrimSpace(event.ToolName) != "" {
		out["tool_name"] = strings.TrimSpace(event.ToolName)
	}
	return out
}

func stringMapToAny(in map[string]string) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func hostHookUnix(t time.Time) int64 {
	if t.IsZero() {
		return time.Now().Unix()
	}
	return t.UTC().Unix()
}

package goncho

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/service/internal/sensitive"
	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

const hostHookPayloadMaxBytes = 16 * 1024

// HostHookEventName is a host-neutral lifecycle/tool event name accepted by
// CaptureHostHook. Host adapters translate Claude Code, Pi, MCP, or other
// runtime-specific hooks into this shape before handing them to Goncho.
type HostHookEventName string

const (
	HostHookSessionStart      HostHookEventName = "session_start"
	HostHookPrompt            HostHookEventName = "prompt"
	HostHookUserPrompt        HostHookEventName = "user_prompt"
	HostHookPreToolUse        HostHookEventName = "pre_tool_use"
	HostHookPostToolUse       HostHookEventName = "post_tool_use"
	HostHookToolFailure       HostHookEventName = "tool_failure"
	HostHookAssistantResponse HostHookEventName = "assistant_response"
	HostHookCompaction        HostHookEventName = "compaction"
	HostHookCompact           HostHookEventName = "compact"
	HostHookSubagentStart     HostHookEventName = "subagent_start"
	HostHookSubagentStop      HostHookEventName = "subagent_stop"
	HostHookStop              HostHookEventName = "stop"
	HostHookSessionEnd        HostHookEventName = "session_end"
	HostHookFailure           HostHookEventName = "failure"
)

// HostHookEventSchema documents the host-neutral JSON event contract adapters
// should emit before calling CaptureHostHook.
type HostHookEventSchema struct {
	Event          HostHookEventName `json:"event"`
	Description    string            `json:"description"`
	RequiredFields []string          `json:"required_fields"`
	JSONSchema     map[string]any    `json:"json_schema"`
}

// HostHookEventSchemas returns the P1 automatic-capture lifecycle event catalog.
func HostHookEventSchemas() []HostHookEventSchema {
	specs := []struct {
		event       HostHookEventName
		description string
		required    []string
	}{
		{HostHookPrompt, "User prompt submitted to the host agent.", []string{"event", "peer_id", "session_key", "content"}},
		{HostHookAssistantResponse, "Assistant response emitted by the host agent.", []string{"event", "peer_id", "session_key", "content"}},
		{HostHookPreToolUse, "Tool call is about to execute.", []string{"event", "peer_id", "session_key", "tool_name", "input"}},
		{HostHookPostToolUse, "Tool call completed successfully or with explicit success status.", []string{"event", "peer_id", "session_key", "tool_name"}},
		{HostHookToolFailure, "Tool call failed and should be captured as error evidence.", []string{"event", "peer_id", "session_key", "tool_name", "error"}},
		{HostHookCompaction, "Host compaction/pre-compact lifecycle event.", []string{"event", "peer_id", "session_key", "summary"}},
		{HostHookSubagentStart, "Subagent worker started under a parent session.", []string{"event", "peer_id", "session_key", "context_id"}},
		{HostHookSubagentStop, "Subagent worker stopped under a parent session.", []string{"event", "peer_id", "session_key", "context_id"}},
		{HostHookStop, "Host stop hook fired before final session closure.", []string{"event", "peer_id", "session_key"}},
		{HostHookSessionEnd, "Host session ended, optionally with summary.", []string{"event", "peer_id", "session_key"}},
	}
	out := make([]HostHookEventSchema, 0, len(specs))
	for _, spec := range specs {
		out = append(out, HostHookEventSchema{
			Event:          spec.event,
			Description:    spec.description,
			RequiredFields: cloneStrings(spec.required),
			JSONSchema:     hostHookJSONSchema(spec.event, spec.required),
		})
	}
	return out
}

func hostHookJSONSchema(event HostHookEventName, required []string) map[string]any {
	return map[string]any{
		"type":     "object",
		"required": cloneStrings(required),
		"properties": map[string]any{
			"event":        map[string]any{"type": "string", "const": string(event)},
			"host":         map[string]any{"type": "string"},
			"workspace_id": map[string]any{"type": "string"},
			"profile_id":   map[string]any{"type": "string"},
			"peer_id":      map[string]any{"type": "string"},
			"session_key":  map[string]any{"type": "string"},
			"context_id":   map[string]any{"type": "string"},
			"tool_name":    map[string]any{"type": "string"},
			"content":      map[string]any{"type": "string"},
			"input":        map[string]any{"type": "string"},
			"output":       map[string]any{"type": "string"},
			"error":        map[string]any{"type": "string"},
			"summary":      map[string]any{"type": "string"},
			"success":      map[string]any{"type": "boolean"},
			"metadata":     map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}},
			"observed_at":  map[string]any{"type": "string", "format": "date-time"},
		},
	}
}

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

type hostHookFilterResult struct {
	Event          HostHookEvent
	Redacted       bool
	RedactionCount int
	Truncated      bool
}

// CaptureHostHook records a host-neutral hook event into Goncho's observation
// log and, for conversational events, the session message stream.
func (s *Service) CaptureHostHook(ctx context.Context, event HostHookEvent) (HookCaptureResult, error) {
	if s == nil {
		return HookCaptureResult{}, fmt.Errorf("goncho: nil service")
	}
	filtered := filterHostHookEvent(event)
	event = filtered.Event
	kind, err := observationKindForHostHook(event)
	if err != nil {
		return HookCaptureResult{}, err
	}
	metadata := hostHookMetadata(event)
	if filtered.Redacted {
		metadata["hook_redacted"] = "true"
		metadata["hook_redaction_count"] = strconv.Itoa(filtered.RedactionCount)
	}
	if filtered.Truncated {
		metadata["hook_truncated"] = "true"
	}
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

var hostHookRedactionRules = []struct {
	kind string
	re   *regexp.Regexp
}{
	{kind: "private", re: regexp.MustCompile(`(?is)<private>.*?</private>`)},
	{kind: "pem_private_key", re: regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----`)},
	{kind: "authorization", re: regexp.MustCompile(`(?i)Authorization:\s*Bearer\s+[^\s\r\n]+`)},
	{kind: "json_secret", re: regexp.MustCompile(`(?i)"([^"]*(?:secret|token|password|api_key|private_key|authorization)[^"]*)"\s*:\s*"[^"]*"`)},
	{kind: "env_secret", re: regexp.MustCompile(`(?im)\b[A-Z0-9_]*(?:SECRET|TOKEN|PASSWORD|API_KEY|PRIVATE_KEY)[A-Z0-9_]*\s*=\s*[^\s\r\n]+`)},
	{kind: "api_key", re: regexp.MustCompile(`\b(?:sk-[A-Za-z0-9_-]+|ghp_[A-Za-z0-9_]+|github_pat_[A-Za-z0-9_]+)\b`)},
}

func filterHostHookEvent(event HostHookEvent) hostHookFilterResult {
	out := hostHookFilterResult{Event: event}
	var filtered string
	filtered, out = filterHostHookString(event.Content, out)
	out.Event.Content = filtered
	filtered, out = filterHostHookString(event.Input, out)
	out.Event.Input = filtered
	filtered, out = filterHostHookString(event.Output, out)
	out.Event.Output = filtered
	filtered, out = filterHostHookString(event.Error, out)
	out.Event.Error = filtered
	filtered, out = filterHostHookString(event.Summary, out)
	out.Event.Summary = filtered
	if event.Metadata != nil {
		out.Event.Metadata = make(map[string]string, len(event.Metadata))
		for key, value := range event.Metadata {
			filteredValue, next := filterHostHookString(value, out)
			out = next
			if hostHookSensitiveKey(key) && filteredValue == value && strings.TrimSpace(value) != "" {
				filteredValue = "[REDACTED:metadata_secret]"
				out.Redacted = true
				out.RedactionCount++
			}
			out.Event.Metadata[key] = filteredValue
		}
	}
	return out
}

func filterHostHookString(value string, state hostHookFilterResult) (string, hostHookFilterResult) {
	value = strings.ToValidUTF8(value, "\uFFFD")
	for _, rule := range hostHookRedactionRules {
		count := 0
		value = rule.re.ReplaceAllStringFunc(value, func(match string) string {
			count++
			if rule.kind == "json_secret" {
				parts := strings.SplitN(match, ":", 2)
				if len(parts) == 2 {
					return parts[0] + `:"[REDACTED:json_secret]"`
				}
			}
			return "[REDACTED:" + rule.kind + "]"
		})
		if count > 0 {
			state.Redacted = true
			state.RedactionCount += count
		}
	}
	if len([]byte(value)) > hostHookPayloadMaxBytes {
		value = textutil.TruncateUTF8Bytes(value, hostHookPayloadMaxBytes)
		state.Truncated = true
	}
	return value, state
}

func hostHookSensitiveKey(key string) bool {
	return sensitive.MetadataKeySecretLike(key)
}

func observationKindForHostHook(event HostHookEvent) (ObservationKind, error) {
	switch event.Event {
	case HostHookSessionStart:
		return ObservationKindSessionStart, nil
	case HostHookPrompt, HostHookUserPrompt:
		return ObservationKindUserPrompt, nil
	case HostHookPreToolUse:
		return ObservationKindToolCall, nil
	case HostHookPostToolUse:
		if hostHookFailed(event) {
			return ObservationKindToolError, nil
		}
		return ObservationKindToolResult, nil
	case HostHookToolFailure:
		return ObservationKindToolError, nil
	case HostHookAssistantResponse:
		return ObservationKindAssistantResponse, nil
	case HostHookCompaction, HostHookCompact:
		return ObservationKindCompact, nil
	case HostHookSubagentStart, HostHookSubagentStop:
		return ObservationKindCustom, nil
	case HostHookStop, HostHookSessionEnd:
		return ObservationKindSessionEnd, nil
	case HostHookFailure:
		return ObservationKindToolError, nil
	default:
		return "", fmt.Errorf("goncho: unsupported host hook event %q", event.Event)
	}
}

func hostHookObservationIO(event HostHookEvent) (string, string) {
	switch event.Event {
	case HostHookPrompt, HostHookUserPrompt:
		return firstNonBlank(event.Content, event.Input), ""
	case HostHookAssistantResponse:
		return event.Input, firstNonBlank(event.Content, event.Output)
	case HostHookStop, HostHookSessionEnd:
		return event.Input, firstNonBlank(event.Summary, event.Output, event.Content)
	default:
		return firstNonBlank(event.Input, event.Content), firstNonBlank(event.Output, event.Error, event.Summary)
	}
}

func hostHookMessageRole(event HostHookEventName) string {
	switch event {
	case HostHookPrompt, HostHookUserPrompt:
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
	case HostHookPreToolUse, HostHookPostToolUse, HostHookToolFailure, HostHookFailure:
		value := !hostHookFailed(event)
		return &value
	default:
		return nil
	}
}

func hostHookFailed(event HostHookEvent) bool {
	return strings.TrimSpace(event.Error) != "" || (event.Success != nil && !*event.Success) || event.Event == HostHookFailure || event.Event == HostHookToolFailure
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
	if event.Event == HostHookSubagentStart || event.Event == HostHookSubagentStop {
		out["custom_kind"] = string(event.Event)
	}
	return out
}

func hostHookUnix(t time.Time) int64 {
	if t.IsZero() {
		return time.Now().Unix()
	}
	return t.UTC().Unix()
}

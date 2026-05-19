package goncho

import (
	"strings"
	"time"
)

type TelemetryMetrics struct {
	InputTokens         int
	OutputTokens        int
	CacheReadTokens     int
	CacheWriteTokens    int
	ReasoningTokens     int
	QueueItemsProcessed int
	ToolCallsCount      int
	ToolErrors          int
	DurationMs          int64
}

type TelemetryEventInput struct {
	Type       string
	Payload    map[string]any
	Timestamp  time.Time
	Workspace  string
	Peer       string
	Session    string
	SessionKey string
	Metrics    TelemetryMetrics
}

type EventMatrixEntry struct {
	Name       string
	Category   string
	Divergence DivergenceEvidence
}

type DivergenceEvidence struct {
	Classification string
	UpstreamRef    string
}

const DivergenceLocal = "local"
const DivergenceOwnedExcluded = "owned_excluded"

type Event struct {
	Name          string         `json:"name"`
	UpstreamEvent string         `json:"upstream_event,omitempty"`
	Source        string         `json:"source"`
	Category      string         `json:"category"`
	EventType     string         `json:"event_type,omitempty"`
	Timestamp     string         `json:"timestamp"`
	Workspace     string         `json:"workspace_id,omitempty"`
	PeerID        string         `json:"peer_id,omitempty"`
	SessionID     string         `json:"session_key,omitempty"`
	Payload       map[string]any `json:"payload,omitempty"`
	Metrics       EventMetrics   `json:"metrics,omitempty"`
	Divergence    *DivergenceEvidence `json:"divergence,omitempty"`
	TokensIn      int            `json:"tokens_in,omitempty"`
	TokensOut     int            `json:"tokens_out,omitempty"`
	DurationMs    int64          `json:"duration_ms,omitempty"`
}

type EventMetrics struct {
	InputTokens         int `json:"input_tokens,omitempty"`
	OutputTokens        int `json:"output_tokens,omitempty"`
	CacheReadTokens     int `json:"cache_read_tokens,omitempty"`
	CacheWriteTokens    int `json:"cache_write_tokens,omitempty"`
	ReasoningTokens     int `json:"reasoning_tokens,omitempty"`
	QueueItemsProcessed int `json:"queue_items_processed,omitempty"`
	ToolCallsCount      int `json:"tool_calls_count,omitempty"`
	ToolErrors          int `json:"tool_errors,omitempty"`
	DurationMs          int64 `json:"duration_ms,omitempty"`
}

type ReasoningTraceInput struct {
	TraceID      string
	TreeNodeID   string
	ParentID     string
	Level        int
	EventType    string
	TaskType     string
	Provider     string
	Model        string
	StartedAt    time.Time
	FinishedAt   time.Time
	InputTokens  int
	OutputTokens int
	Prompt       string
	Response     string
	Workspace    string
	Peer         string
	Session      string
	ToolCalls    []string
	Timestamp    time.Time
	DurationMs   int64
	Payload      map[string]any
}

type ReasoningTraceRecord struct {
	TraceID      string   `json:"trace_id"`
	TreeNodeID   string   `json:"tree_node_id"`
	ParentID     string   `json:"parent_id"`
	Level        int      `json:"level"`
	EventType    string   `json:"event_type"`
	TaskType     string   `json:"task_type"`
	Provider     string   `json:"provider"`
	Model        string   `json:"model"`
	InputTokens  int      `json:"input_tokens"`
	OutputTokens int      `json:"output_tokens"`
	DurationMs   int64    `json:"duration_ms"`
	Prompt       string   `json:"prompt"`
	Response     string   `json:"response"`
	Workspace    string   `json:"workspace_id"`
	Peer         string   `json:"peer_id"`
	Session      string   `json:"session_key"`
	ToolCalls    []string `json:"tool_calls"`
	Timestamp    string   `json:"timestamp"`
}

func (r *ReasoningTraceRecord) TelemetryEvent() Event {
	return Event{
		Name:      "gormes.goncho.reasoning_trace",
		Category:  "reasoning",
		EventType: r.EventType,
	}
}

var eventMatrix = map[string]EventMatrixEntry{
	"representation.completed": {
		Name:       "gormes.goncho.representation.completed",
		Category:   "representation",
		Divergence: DivergenceEvidence{Classification: DivergenceLocal, UpstreamRef: "honcho.representation.completed"},
	},
	"honcho.sentry.trace": {
		Name:       "gormes.telemetry.exporter.excluded",
		Category:   "telemetry",
		Divergence: DivergenceEvidence{Classification: DivergenceOwnedExcluded, UpstreamRef: "honcho.sentry.trace"},
	},
}

func LookupTelemetryEvent(eventType string) (EventMatrixEntry, bool) {
	entry, ok := eventMatrix[eventType]
	return entry, ok
}

func SummarizePayload(payload map[string]any) (string, error) {
	return "", nil
}

func NormalizeEvent(event Event, timestamp time.Time) Event {
	event.Timestamp = timestamp.UTC().Format(time.RFC3339)
	return event
}

func NewReasoningTraceRecord(input ReasoningTraceInput) ReasoningTraceRecord {
	return ReasoningTraceRecord{
		TraceID:      strings.TrimSpace(input.TraceID),
		TreeNodeID:   strings.TrimSpace(input.TreeNodeID),
		ParentID:     strings.TrimSpace(input.ParentID),
		Level:        input.Level,
		EventType:    input.EventType,
		TaskType:     input.TaskType,
		Provider:     input.Provider,
		Model:        input.Model,
		InputTokens:  input.InputTokens,
		OutputTokens: input.OutputTokens,
		DurationMs:   input.DurationMs,
		Prompt:       strings.TrimSpace(input.Prompt),
		Response:     strings.TrimSpace(input.Response),
		Workspace:    strings.TrimSpace(input.Workspace),
		Peer:         strings.TrimSpace(input.Peer),
		Session:      strings.TrimSpace(input.Session),
		ToolCalls:    input.ToolCalls,
		Timestamp:    input.Timestamp.UTC().Format(time.RFC3339),
	}
}

func NewTelemetryEvent(input TelemetryEventInput) Event {
	session := input.Session
	if session == "" {
		session = input.SessionKey
	}
	entry, ok := LookupTelemetryEvent(input.Type)
	if !ok {
		entry = EventMatrixEntry{
			Name:     input.Type,
			Category: "goncho",
			Divergence: DivergenceEvidence{
				Classification: DivergenceLocal,
			},
		}
	}
	summary, _ := SummarizePayload(input.Payload)
	div := entry.Divergence
	event := Event{
		Name:          entry.Name,
		UpstreamEvent: input.Type,
		Source:        "honcho",
		Category:      entry.Category,
		Workspace:     strings.TrimSpace(input.Workspace),
		PeerID:        strings.TrimSpace(input.Peer),
		SessionID:     strings.TrimSpace(session),
		Payload:       input.Payload,
		Metrics: EventMetrics{
			InputTokens:         input.Metrics.InputTokens,
			OutputTokens:        input.Metrics.OutputTokens,
			CacheReadTokens:     input.Metrics.CacheReadTokens,
			CacheWriteTokens:    input.Metrics.CacheWriteTokens,
			ReasoningTokens:     input.Metrics.ReasoningTokens,
			QueueItemsProcessed: input.Metrics.QueueItemsProcessed,
			ToolCallsCount:      input.Metrics.ToolCallsCount,
			ToolErrors:          input.Metrics.ToolErrors,
			DurationMs:          input.Metrics.DurationMs,
		},
		TokensIn:   input.Metrics.InputTokens,
		TokensOut:  input.Metrics.OutputTokens,
		DurationMs: input.Metrics.DurationMs,
	}
	if div.Classification != "" {
		event.Divergence = &div
	}
	if summary != "" {
		if event.Payload == nil {
			event.Payload = make(map[string]any)
		}
		event.Payload["summary"] = summary
	}
	return NormalizeEvent(event, input.Timestamp)
}

package telemetry

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
	Workspace  string
	SessionKey string
	Peer       string
	RunID      string
	AgentID    string
	ResourceID string
	Iteration  int
	Timestamp  time.Time
	Metrics    TelemetryMetrics
	Payload    map[string]any
}

func NewTelemetryEvent(input TelemetryEventInput) Event {
	entry, ok := LookupTelemetryEvent(input.Type)
	if !ok {
		entry = EventMatrixEntry{
			UpstreamEvent: strings.TrimSpace(input.Type),
			LocalEvent:    "gormes.goncho." + strings.ReplaceAll(strings.ToLower(strings.TrimSpace(input.Type)), " ", "_"),
			Source:        "honcho",
			Category:      "unknown",
			Divergence: DivergenceEvidence{
				Classification: DivergenceLocal,
				Replacement:    "redacted local telemetry, audit, and insights evidence",
			},
		}
	}
	summary, _ := SummarizePayload(input.Payload)
	event := Event{
		Name:             entry.LocalEvent,
		UpstreamEvent:    entry.UpstreamEvent,
		Source:           entry.Source,
		Category:         entry.Category,
		Timestamp:        input.Timestamp,
		SessionID:        strings.TrimSpace(input.SessionKey),
		AgentID:          strings.TrimSpace(input.AgentID),
		PeerID:           strings.TrimSpace(input.Peer),
		WorkspaceID:      strings.TrimSpace(input.Workspace),
		RunID:            strings.TrimSpace(input.RunID),
		EventType:        strings.TrimSpace(input.Type),
		TokensIn:         nonNegativeMetric(input.Metrics.InputTokens),
		TokensOut:        nonNegativeMetric(input.Metrics.OutputTokens),
		CacheReadTokens:  nonNegativeMetric(input.Metrics.CacheReadTokens),
		CacheWriteTokens: nonNegativeMetric(input.Metrics.CacheWriteTokens),
		ReasoningTokens:  nonNegativeMetric(input.Metrics.ReasoningTokens),
		ToolCalls:        nonNegativeMetric(input.Metrics.ToolCallsCount),
		ToolErrors:       nonNegativeMetric(input.Metrics.ToolErrors),
		QueueItems:       nonNegativeMetric(input.Metrics.QueueItemsProcessed),
		DurationMs:       input.Metrics.DurationMs,
		PayloadSummary:   summary,
	}
	if entry.Divergence.Classification != "" {
		divergence := entry.Divergence
		event.Divergence = &divergence
	}
	return NormalizeEvent(event, input.Timestamp)
}

package telemetry

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestGonchoTelemetryMapsHonchoEventNamesToLocalEvidence(t *testing.T) {
	now := time.Date(2026, 4, 29, 14, 0, 0, 0, time.UTC)

	event := NewTelemetryEvent(TelemetryEventInput{
		Type:       "representation.completed",
		Workspace:  "workspace-a",
		SessionKey: "session-a",
		Peer:       "alice",
		Timestamp:  now,
		Metrics: TelemetryMetrics{
			InputTokens:         100,
			OutputTokens:        30,
			QueueItemsProcessed: 3,
			DurationMs:          250,
		},
		Payload: map[string]any{
			"prompt":  "raw prompt with sk-secret",
			"api_key": "sk-secret",
		},
	})

	if event.Name != "gormes.goncho.representation.completed" || event.UpstreamEvent != "representation.completed" {
		t.Fatalf("event names = %q/%q, want representation mapping", event.Name, event.UpstreamEvent)
	}
	if event.Source != "honcho" || event.Category != "representation" {
		t.Fatalf("event source/category = %q/%q, want honcho/representation", event.Source, event.Category)
	}
	if event.SessionID != "session-a" || event.PeerID != "alice" {
		t.Fatalf("event session/peer = %q/%q, want session-a/alice", event.SessionID, event.PeerID)
	}
	if event.TokensIn != 100 || event.TokensOut != 30 || event.DurationMs != 250 {
		t.Fatalf("event metrics = %+v, want token/duration metrics", event)
	}
	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Marshal(event): %v", err)
	}
	for _, forbidden := range []string{"raw prompt", "sk-secret"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("event leaked %q in %s", forbidden, raw)
		}
	}

	exporter := NewTelemetryEvent(TelemetryEventInput{Type: "honcho.sentry.trace", Timestamp: now})
	if exporter.Name != "gormes.telemetry.exporter.excluded" || exporter.Divergence == nil {
		t.Fatalf("exporter mapping = %+v, want excluded divergence evidence", exporter)
	}
	if exporter.Divergence.Classification != DivergenceOwnedExcluded {
		t.Fatalf("exporter classification = %q, want owned_excluded", exporter.Divergence.Classification)
	}
}

func TestGonchoTelemetryReasoningTracePreservesTreeWithoutRawPayload(t *testing.T) {
	started := time.Date(2026, 4, 29, 14, 10, 0, 0, time.UTC)

	trace := NewReasoningTraceRecord(ReasoningTraceInput{
		TraceID:      "trace-goncho",
		TreeNodeID:   "node-child",
		ParentID:     "node-parent",
		Level:        1,
		EventType:    "dream.specialist",
		TaskType:     "dream",
		Provider:     "anthropic",
		Model:        "claude-opus",
		StartedAt:    started,
		FinishedAt:   started.Add(100 * time.Millisecond),
		InputTokens:  80,
		OutputTokens: 25,
		Payload: map[string]any{
			"messages": []map[string]string{{"role": "user", "content": "private dream prompt"}},
			"token":    "provider-token",
		},
	})

	if trace.TraceID != "trace-goncho" || trace.TreeNodeID != "node-child" || trace.ParentID != "node-parent" || trace.Level != 1 {
		t.Fatalf("trace shape = %+v, want preserved tree ids", trace)
	}
	event := trace.TelemetryEvent()
	if event.Name != "gormes.goncho.reasoning_trace" || event.EventType != "dream.specialist" {
		t.Fatalf("trace event = %+v, want local reasoning trace for dream.specialist", event)
	}
	raw, err := json.Marshal(trace)
	if err != nil {
		t.Fatalf("Marshal(trace): %v", err)
	}
	for _, forbidden := range []string{"private dream prompt", "provider-token"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("trace leaked %q in %s", forbidden, raw)
		}
	}
}

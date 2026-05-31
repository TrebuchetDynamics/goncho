package goncho

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/goncho/service/internal/ptrutil"
)

func TestHostHookEventSchemasCoverP1AgentLifecycleEvents(t *testing.T) {
	want := []HostHookEventName{
		HostHookPrompt,
		HostHookAssistantResponse,
		HostHookPreToolUse,
		HostHookPostToolUse,
		HostHookToolFailure,
		HostHookCompaction,
		HostHookSubagentStart,
		HostHookSubagentStop,
		HostHookStop,
		HostHookSessionEnd,
	}
	schemas := HostHookEventSchemas()
	if len(schemas) != len(want) {
		t.Fatalf("schemas len = %d, want %d: %+v", len(schemas), len(want), schemas)
	}
	byEvent := map[HostHookEventName]HostHookEventSchema{}
	for _, schema := range schemas {
		byEvent[schema.Event] = schema
		if schema.JSONSchema["type"] != "object" {
			t.Fatalf("schema %s JSONSchema = %+v, want object schema", schema.Event, schema.JSONSchema)
		}
	}
	for _, event := range want {
		schema, ok := byEvent[event]
		if !ok {
			t.Fatalf("missing schema for %s", event)
		}
		if schema.Description == "" || len(schema.RequiredFields) == 0 {
			t.Fatalf("schema %s = %+v, want description and required fields", event, schema)
		}
	}
}

func TestServiceCaptureHostHookAcceptsP1AgentLifecycleEvents(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	ctx := context.Background()
	for _, event := range []HostHookEvent{
		{Event: HostHookPrompt, PeerID: "user-ana", SessionKey: "sess-p1", Content: "Please run the hook schema slice."},
		{Event: HostHookAssistantResponse, PeerID: "agent-pi", SessionKey: "sess-p1", Content: "I will implement host schemas."},
		{Event: HostHookPreToolUse, PeerID: "agent-pi", SessionKey: "sess-p1", ToolName: "bash", Input: "go test ./service"},
		{Event: HostHookPostToolUse, PeerID: "agent-pi", SessionKey: "sess-p1", ToolName: "bash", Output: "ok"},
		{Event: HostHookToolFailure, PeerID: "agent-pi", SessionKey: "sess-p1", ToolName: "bash", Error: "exit status 1"},
		{Event: HostHookCompaction, PeerID: "agent-pi", SessionKey: "sess-p1", Summary: "Compacted hook context."},
		{Event: HostHookSubagentStart, PeerID: "agent-pi", SessionKey: "sess-p1", ContextID: "sub-1", Content: "Subagent started."},
		{Event: HostHookSubagentStop, PeerID: "agent-pi", SessionKey: "sess-p1", ContextID: "sub-1", Content: "Subagent stopped."},
		{Event: HostHookStop, PeerID: "agent-pi", SessionKey: "sess-p1", Summary: "Stop hook fired."},
		{Event: HostHookSessionEnd, PeerID: "agent-pi", SessionKey: "sess-p1", Summary: "Session ended."},
	} {
		if _, err := svc.CaptureHostHook(ctx, event); err != nil {
			t.Fatalf("CaptureHostHook(%s): %v", event.Event, err)
		}
	}
	list, err := svc.ListObservations(ctx, ObservationQuery{SessionKey: "sess-p1", Limit: 20})
	if err != nil {
		t.Fatalf("ListObservations: %v", err)
	}
	if list.Count != 10 {
		t.Fatalf("observations = %d, want 10", list.Count)
	}
}

func TestServiceCaptureHostHookRedactsSecretsAndTruncatesBeforeStorage(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	secretContent := "Authorization: Bearer secret-token\nOPENAI_API_KEY=sk-live-secret\n" + strings.Repeat("x", hostHookPayloadMaxBytes+128)
	result, err := svc.CaptureHostHook(context.Background(), HostHookEvent{
		Event:      HostHookPrompt,
		PeerID:     "user-redact",
		SessionKey: "sess-redact",
		Content:    secretContent,
		Metadata:   map[string]string{"api_token": "secret-token"},
	})
	if err != nil {
		t.Fatalf("CaptureHostHook: %v", err)
	}
	if len(result.Messages) != 1 || len(result.Observations) != 1 {
		t.Fatalf("result = %+v, want one message and one observation", result)
	}
	stored := result.Messages[0].Content + result.Observations[0].Input + result.Observations[0].Metadata["api_token"]
	for _, leaked := range []string{"secret-token", "sk-live-secret"} {
		if strings.Contains(stored, leaked) {
			t.Fatalf("stored hook payload leaked %q in %.200q", leaked, stored)
		}
	}
	if !strings.Contains(stored, "[REDACTED:authorization]") || !strings.Contains(stored, "[REDACTED:env_secret]") {
		t.Fatalf("stored hook payload missing redaction markers in %.200q", stored)
	}
	if len([]byte(result.Messages[0].Content)) > hostHookPayloadMaxBytes {
		t.Fatalf("message content bytes = %d, want <= %d", len([]byte(result.Messages[0].Content)), hostHookPayloadMaxBytes)
	}
	if result.Messages[0].Metadata["hook_redacted"] != "true" || result.Messages[0].Metadata["hook_truncated"] != "true" {
		t.Fatalf("message metadata = %+v, want hook redaction/truncation evidence", result.Messages[0].Metadata)
	}
}

func TestServiceCaptureHostHookMapsToObserveMessagesAndSummary(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	ctx := context.Background()
	observedAt := time.Date(2026, 5, 24, 10, 0, 0, 0, time.UTC)
	events := []HostHookEvent{
		{
			Event:      HostHookUserPrompt,
			PeerID:     "user-ana",
			SessionKey: "sess-hooks",
			Content:    "Please inspect service/hook_capture.go and run tests.",
			ObservedAt: observedAt,
		},
		{
			Event:      HostHookPostToolUse,
			PeerID:     "agent-pi",
			SessionKey: "sess-hooks",
			ToolName:   "bash",
			Input:      "go test ./service -run HookCapture",
			Output:     "build failed: missing HostHookEvent",
			Success:    ptrutil.Bool(false),
			ObservedAt: observedAt.Add(time.Second),
		},
		{
			Event:      HostHookAssistantResponse,
			PeerID:     "agent-pi",
			SessionKey: "sess-hooks",
			Content:    "Implemented host-neutral hook capture. Next: run go test ./...",
			ObservedAt: observedAt.Add(2 * time.Second),
		},
		{
			Event:      HostHookSessionEnd,
			PeerID:     "agent-pi",
			SessionKey: "sess-hooks",
			Summary:    "Hook capture session ended after wiring Observe, CreateMessages, and summary persistence.",
			ObservedAt: observedAt.Add(3 * time.Second),
		},
	}

	var aggregate HookCaptureResult
	for _, event := range events {
		result, err := svc.CaptureHostHook(ctx, event)
		if err != nil {
			t.Fatalf("CaptureHostHook(%s): %v", event.Event, err)
		}
		aggregate.Observations = append(aggregate.Observations, result.Observations...)
		aggregate.Messages = append(aggregate.Messages, result.Messages...)
	}

	if len(aggregate.Observations) != 4 {
		t.Fatalf("observations = %d, want 4", len(aggregate.Observations))
	}
	if aggregate.Observations[1].Kind != ObservationKindToolError {
		t.Fatalf("tool observation kind = %s, want tool_error", aggregate.Observations[1].Kind)
	}
	if aggregate.Observations[1].Success == nil || *aggregate.Observations[1].Success {
		t.Fatalf("tool observation success = %v, want false", aggregate.Observations[1].Success)
	}
	if aggregate.Observations[1].Metadata["hook_event"] != string(HostHookPostToolUse) || aggregate.Observations[1].Metadata["tool_name"] != "bash" {
		t.Fatalf("tool metadata = %+v, want hook provenance and tool name", aggregate.Observations[1].Metadata)
	}
	if len(aggregate.Messages) != 2 {
		t.Fatalf("messages = %d, want prompt + assistant response", len(aggregate.Messages))
	}
	if aggregate.Messages[0].Role != "user" || aggregate.Messages[0].Content != events[0].Content {
		t.Fatalf("first message = %+v, want captured user prompt", aggregate.Messages[0])
	}
	if aggregate.Messages[1].Role != "assistant" || aggregate.Messages[1].Content != events[2].Content {
		t.Fatalf("second message = %+v, want captured assistant response", aggregate.Messages[1])
	}

	list, err := svc.ListObservations(ctx, ObservationQuery{SessionKey: "sess-hooks", Limit: 10})
	if err != nil {
		t.Fatalf("ListObservations: %v", err)
	}
	if list.Count != 4 {
		t.Fatalf("listed observations = %d, want 4", list.Count)
	}
	summary, err := getSessionSummary(ctx, svc.db, "default", "sess-hooks", "short")
	if err != nil {
		t.Fatalf("get summary: %v", err)
	}
	if summary == nil || summary.Content != events[3].Summary {
		t.Fatalf("summary = %+v, want host session summary", summary)
	}
}

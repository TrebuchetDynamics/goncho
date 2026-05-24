package goncho

import (
	"context"
	"testing"
	"time"
)

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
			Success:    boolPtr(false),
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

func boolPtr(v bool) *bool { return &v }

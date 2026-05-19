package goncho

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestContractChatParamsJSONShape(t *testing.T) {
	raw, err := json.Marshal(ChatParams{
		Query:          "What does Juan prefer?",
		SessionID:      "session-123",
		Target:         "assistant",
		ReasoningLevel: "high",
		Stream:         true,
	})
	if err != nil {
		t.Fatal(err)
	}

	text := string(raw)
	for _, want := range []string{
		`"query":"What does Juan prefer?"`,
		`"session_id":"session-123"`,
		`"target":"assistant"`,
		`"reasoning_level":"high"`,
		`"stream":true`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("ChatParams JSON missing %s in %s", want, raw)
		}
	}
}

func TestService_ChatDefaultsReasoningAndReturnsContentOnlyResponse(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	if _, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "telegram:6586915095",
		Conclusion: "The user prefers exact evidence-first reports.",
		SessionKey: "session-123",
	}); err != nil {
		t.Fatal(err)
	}

	got, err := svc.Chat(ctx, "telegram:6586915095", ChatParams{
		Query:     "How should I answer?",
		SessionID: "session-123",
	})
	if err != nil {
		t.Fatal(err)
	}

	var shape map[string]string
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &shape); err != nil {
		t.Fatalf("ChatResult JSON should be content-only string fields, got %s: %v", raw, err)
	}
	if len(shape) != 1 {
		t.Fatalf("ChatResult JSON = %s, want only content", raw)
	}
	content := shape["content"]
	if content == "" {
		t.Fatalf("ChatResult JSON = %s, want non-empty content", raw)
	}
	for _, want := range []string{
		"Query: How should I answer?",
		"Reasoning level: low",
		"The user prefers exact evidence-first reports.",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("ChatResult content missing %q in %q", want, content)
		}
	}
}

func TestService_ChatRejectsInvalidReasoningLevel(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	_, err := svc.Chat(context.Background(), "telegram:6586915095", ChatParams{
		Query:          "How should I answer?",
		ReasoningLevel: "turbo",
	})
	if err == nil {
		t.Fatal("expected invalid reasoning_level error")
	}
	if !strings.Contains(err.Error(), "reasoning_level") {
		t.Fatalf("error = %v, want reasoning_level validation", err)
	}
}

func TestService_ChatReportsStreamingAndTargetDegradationInContent(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	got, err := svc.Chat(context.Background(), "telegram:6586915095", ChatParams{
		Query:          "What does Juan know about the assistant?",
		Target:         "assistant",
		ReasoningLevel: "medium",
		Stream:         true,
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"Unsupported evidence:",
		"field=stream",
		"field=target",
		"Reasoning level: medium",
	} {
		if !strings.Contains(got.Content, want) {
			t.Fatalf("ChatResult content missing %q in %q", want, got.Content)
		}
	}
}

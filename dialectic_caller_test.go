package goncho

import (
	"context"
	"io"
	"strings"
	"testing"
)

// mockChatStream implements ChatStream for testing.
type mockChatStream struct {
	events []ChatStreamEvent
	idx    int
}

func (m *mockChatStream) Recv(ctx context.Context) (ChatStreamEvent, error) {
	if m.idx >= len(m.events) {
		return ChatStreamEvent{}, io.EOF
	}
	e := m.events[m.idx]
	m.idx++
	return e, nil
}

func (m *mockChatStream) Close() error { return nil }

// mockClient implements the Client interface for testing.
type mockClient struct {
	stream   ChatStream
	opened   bool
	model    string
	system   string
	query    string
}

func (m *mockClient) OpenStream(ctx context.Context, model, systemPrompt, query string) (ChatStream, error) {
	m.opened = true
	m.model = model
	m.system = systemPrompt
	m.query = query
	return m.stream, nil
}

func TestHermesDialecticCaller_StreamsLLMAnswerAndSendsContextPrompt(t *testing.T) {
	client := &mockClient{
		stream: &mockChatStream{
			events: []ChatStreamEvent{
				{Kind: "token", Token: "Use "},
				{Kind: "token", Token: "exact evidence."},
				{Kind: "done", Token: ""},
			},
		},
	}

	caller := NewHermesDialecticCaller(client, "gpt-test")
	answer, err := caller.Chat(context.Background(), "telegram:6586915095", "## Peer Representation\nPrefers exact evidence.", "How should I answer?")
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}
	if answer != "Use exact evidence." {
		t.Fatalf("answer = %q, want streamed token concatenation", answer)
	}

	if !client.opened {
		t.Fatal("dialectic caller must open a stream through the native client")
	}
	if client.model != "gpt-test" {
		t.Fatalf("model = %q, want gpt-test", client.model)
	}
	if !strings.Contains(client.system, "Prefers exact evidence") {
		t.Fatalf("system prompt = %q, want context prompt", client.system)
	}
	if client.query != "How should I answer?" {
		t.Fatalf("query = %q, want user query", client.query)
	}
}

func TestHermesDialecticCaller_PropagatesProviderFailure(t *testing.T) {
	client := &mockClient{
		stream: &mockChatStream{events: []ChatStreamEvent{}},
	}

	caller := NewHermesDialecticCaller(client, "gpt-test")
	_, err := caller.Chat(context.Background(), "user", "context", "query")
	if err == nil {
		t.Fatal("Chat error = nil, want provider empty-stream error")
	}
	if !strings.Contains(err.Error(), "no dialectic answer") {
		t.Fatalf("error = %v, want no dialectic answer evidence", err)
	}
}

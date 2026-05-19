package goncho

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
)

// ChatStreamEvent is one event from a streaming chat response.
type ChatStreamEvent struct {
	Kind  string
	Token string
}

// ChatStream is a streaming chat response reader.
type ChatStream interface {
	Recv(ctx context.Context) (ChatStreamEvent, error)
	Close() error
}

// Client is the minimal chat client interface for dialectic calls.
type Client interface {
	OpenStream(ctx context.Context, model, systemPrompt, query string) (ChatStream, error)
}

// HermesDialecticCaller adapts a native provider client to the Goncho
// DialecticCaller seam. It keeps honcho_reasoning fully in-process.
type HermesDialecticCaller struct {
	client Client
	model  string
}

// NewHermesDialecticCaller returns a DialecticCaller backed by a chat client.
func NewHermesDialecticCaller(client Client, model string) *HermesDialecticCaller {
	return &HermesDialecticCaller{client: client, model: strings.TrimSpace(model)}
}

// Chat sends the supplied Goncho context prompt and query through the native
// provider client, collecting streamed text tokens into one synthesized answer.
func (c *HermesDialecticCaller) Chat(ctx context.Context, peer string, systemPrompt string, query string) (string, error) {
	if c == nil || c.client == nil {
		return "", errors.New("goncho: dialectic caller client is nil")
	}
	stream, err := c.client.OpenStream(ctx, c.model, systemPrompt, query)
	if err != nil {
		return "", fmt.Errorf("goncho: dialectic provider stream: %w", err)
	}
	defer stream.Close()

	var answer strings.Builder
	for {
		ev, err := stream.Recv(ctx)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", fmt.Errorf("goncho: dialectic provider recv: %w", err)
		}
		if ev.Kind == "token" {
			answer.WriteString(ev.Token)
		}
	}
	out := strings.TrimSpace(answer.String())
	if out == "" {
		return "", errors.New("goncho: no dialectic answer from provider")
	}
	return out, nil
}

func dialecticSessionID(peer string) string {
	peer = strings.TrimSpace(peer)
	if peer == "" {
		peer = "unknown"
	}
	return "goncho-dialectic:" + peer
}

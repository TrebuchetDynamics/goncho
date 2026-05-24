package goncho

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ChatCompletionMetadata carries terminal stream metadata that can be attached
// after the assistant response is complete.
type ChatCompletionMetadata struct {
	TokensIn  int `json:"tokens_in,omitempty"`
	TokensOut int `json:"tokens_out,omitempty"`
}

// StreamingChatPersistence buffers stream chunks until a terminal event decides
// whether the assistant response is complete enough to become durable memory.
type StreamingChatPersistence struct {
	service     *Service
	peer        string
	sessionID   string
	chunks      []string
	completed   bool
	interrupted bool
	content     string
}

func (s *Service) NewStreamingChatPersistence(peer string, params ChatParams) (*StreamingChatPersistence, error) {
	peer = strings.TrimSpace(peer)
	if peer == "" {
		return nil, fmt.Errorf("goncho: peer is required")
	}
	return &StreamingChatPersistence{
		service:   s,
		peer:      peer,
		sessionID: strings.TrimSpace(params.SessionID),
	}, nil
}

func (p *StreamingChatPersistence) AppendChunk(chunk string) {
	if p == nil || p.completed || p.interrupted || chunk == "" {
		return
	}
	p.chunks = append(p.chunks, chunk)
}

func (p *StreamingChatPersistence) Complete(ctx context.Context, meta ChatCompletionMetadata) (ChatResult, error) {
	if p == nil || p.service == nil {
		return ChatResult{}, fmt.Errorf("goncho: streaming chat persistence is unavailable")
	}
	if p.interrupted {
		return ChatResult{}, fmt.Errorf("goncho: streaming chat was interrupted")
	}
	if p.completed {
		return ChatResult{Content: p.content}, nil
	}

	content := strings.Join(p.chunks, "")
	metaJSON, err := completionMetadataJSON(meta)
	if err != nil {
		return ChatResult{}, err
	}
	if err := insertAssistantChatTurn(ctx, p.service.db, p.sessionID, p.peer, content, metaJSON); err != nil {
		return ChatResult{}, err
	}
	p.content = content
	p.completed = true
	return ChatResult{Content: content}, nil
}

func (p *StreamingChatPersistence) Interrupt(reason string) ChatResult {
	if p == nil {
		return ChatResult{}
	}
	if p.completed {
		return ChatResult{Content: p.content}
	}
	p.interrupted = true
	p.chunks = nil
	return ChatResult{Content: streamingInterruptedContent(reason)}
}

func completionMetadataJSON(meta ChatCompletionMetadata) (string, error) {
	if meta.TokensIn <= 0 && meta.TokensOut <= 0 {
		return "", nil
	}
	payload := map[string]int{}
	if meta.TokensIn > 0 {
		payload["tokens_in"] = meta.TokensIn
	}
	if meta.TokensOut > 0 {
		payload["tokens_out"] = meta.TokensOut
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("goncho: marshal chat completion metadata: %w", err)
	}
	return string(raw), nil
}

func streamingInterruptedContent(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "interrupted"
	}
	return "Unsupported evidence:\n- field=stream capability=streaming_chat_interrupted reason=" + reason + "; partial assistant content was discarded"
}

package goncho

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"
)

func TestService_ChatStreamDegradedPersistsFinalAssistantResponseOnce(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	peer := "telegram:6586915095"
	sessionID := "sess-stream-chat"

	got, err := svc.Chat(ctx, peer, ChatParams{
		Query:     "What should the assistant remember?",
		SessionID: sessionID,
		Stream:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.Content, "field=stream") {
		t.Fatalf("Chat content missing streaming degradation evidence: %q", got.Content)
	}

	row := readAssistantTurn(t, svc.db, sessionID)
	if row.Count != 1 {
		t.Fatalf("assistant turns for session = %d, want exactly 1", row.Count)
	}
	if row.Role != "assistant" {
		t.Fatalf("role = %q, want assistant", row.Role)
	}
	if row.Content != got.Content {
		t.Fatalf("stored content = %q, want final response %q", row.Content, got.Content)
	}
	if row.ChatID != peer {
		t.Fatalf("chat_id = %q, want assistant peer %q", row.ChatID, peer)
	}
	if row.MemorySyncStatus != "ready" {
		t.Fatalf("memory_sync_status = %q, want ready", row.MemorySyncStatus)
	}
}

func TestStreamingChatPersistenceAccumulatesChunksBeforeSingleWrite(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	peer := "telegram:6586915095"
	sessionID := "sess-stream-complete"

	stream, err := svc.NewStreamingChatPersistence(peer, ChatParams{SessionID: sessionID})
	if err != nil {
		t.Fatal(err)
	}
	stream.AppendChunk("First chunk ")
	stream.AppendChunk("second chunk")

	if row := readAssistantTurn(t, svc.db, sessionID); row.Count != 0 {
		t.Fatalf("assistant turns before completion = %d, want 0", row.Count)
	}
	if countTurnsWithContent(t, svc.db, "First chunk ") != 0 {
		t.Fatal("partial stream chunk was written before completion")
	}

	got, err := stream.Complete(ctx, ChatCompletionMetadata{
		TokensIn:  7,
		TokensOut: 11,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "First chunk second chunk" {
		t.Fatalf("completed content = %q, want accumulated chunks", got.Content)
	}

	row := readAssistantTurn(t, svc.db, sessionID)
	if row.Count != 1 {
		t.Fatalf("assistant turns after completion = %d, want exactly 1", row.Count)
	}
	if row.Content != got.Content {
		t.Fatalf("stored content = %q, want %q", row.Content, got.Content)
	}
	if row.ChatID != peer {
		t.Fatalf("chat_id = %q, want %q", row.ChatID, peer)
	}

	var meta map[string]int
	if err := json.Unmarshal([]byte(row.MetaJSON), &meta); err != nil {
		t.Fatalf("meta_json should contain token metadata: %q: %v", row.MetaJSON, err)
	}
	if meta["tokens_in"] != 7 || meta["tokens_out"] != 11 {
		t.Fatalf("token metadata = %+v, want tokens_in=7 tokens_out=11", meta)
	}

	again, err := stream.Complete(ctx, ChatCompletionMetadata{TokensIn: 99, TokensOut: 99})
	if err != nil {
		t.Fatal(err)
	}
	if again.Content != got.Content {
		t.Fatalf("second complete content = %q, want original %q", again.Content, got.Content)
	}
	if row := readAssistantTurn(t, svc.db, sessionID); row.Count != 1 {
		t.Fatalf("assistant turns after second completion = %d, want still exactly 1", row.Count)
	}
}

func TestStreamingChatPersistenceInterruptRecordsEvidenceWithoutFlushingPartial(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	sessionID := "sess-stream-interrupted"

	stream, err := svc.NewStreamingChatPersistence("telegram:6586915095", ChatParams{SessionID: sessionID})
	if err != nil {
		t.Fatal(err)
	}
	stream.AppendChunk("partial assistant draft")

	got := stream.Interrupt("client_disconnect")
	for _, want := range []string{
		"Unsupported evidence:",
		"field=stream",
		"capability=streaming_chat_interrupted",
		"client_disconnect",
	} {
		if !strings.Contains(got.Content, want) {
			t.Fatalf("interruption result missing %q in %q", want, got.Content)
		}
	}

	if _, err := stream.Complete(ctx, ChatCompletionMetadata{}); err == nil {
		t.Fatal("expected interrupted stream completion to fail")
	}
	if row := readAssistantTurn(t, svc.db, sessionID); row.Count != 0 {
		t.Fatalf("assistant turns after interruption = %d, want 0", row.Count)
	}
	if countTurnsWithContent(t, svc.db, "partial assistant draft") != 0 {
		t.Fatal("partial interrupted stream content was written to memory")
	}
}

type assistantTurnRow struct {
	Count            int
	Role             string
	Content          string
	ChatID           string
	MemorySyncStatus string
	MetaJSON         string
}

func readAssistantTurn(t *testing.T, db *sql.DB, sessionID string) assistantTurnRow {
	t.Helper()

	var row assistantTurnRow
	err := db.QueryRow(`
		SELECT COUNT(*), COALESCE(MAX(role), ''), COALESCE(MAX(content), ''),
		       COALESCE(MAX(chat_id), ''), COALESCE(MAX(memory_sync_status), ''),
		       COALESCE(MAX(meta_json), '')
		FROM turns
		WHERE session_id = ? AND role = 'assistant'
	`, sessionID).Scan(
		&row.Count,
		&row.Role,
		&row.Content,
		&row.ChatID,
		&row.MemorySyncStatus,
		&row.MetaJSON,
	)
	if err != nil {
		t.Fatalf("read assistant turn: %v", err)
	}
	return row
}

func countTurnsWithContent(t *testing.T, db *sql.DB, content string) int {
	t.Helper()

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM turns WHERE content = ?`, content).Scan(&count); err != nil {
		t.Fatalf("count turns with content: %v", err)
	}
	return count
}

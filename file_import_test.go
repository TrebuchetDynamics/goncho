package goncho

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestService_ImportFilePublicFacadeCreatesSessionMessageWithEvidence(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	createdAt := time.Unix(1_714_558_400, 0).UTC()
	got, err := svc.ImportFile(context.Background(), ImportFileParams{
		SessionKey:  "session-import-public",
		PeerID:      "telegram:6586915095",
		Filename:    "MEMORY.md",
		ContentType: "text/markdown",
		Content:     []byte("# Memory\n\nJuan prefers evidence-first reports."),
		CreatedAt:   &createdAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.WorkspaceID != svc.workspaceID || got.SessionKey != "session-import-public" || got.PeerID != "telegram:6586915095" {
		t.Fatalf("result identity = %+v, want public facade import identity", got)
	}
	if len(got.Messages) != 1 || got.Messages[0].Role != "user" || !strings.Contains(got.Messages[0].Content, "evidence-first") {
		t.Fatalf("messages = %+v, want imported user session message", got.Messages)
	}
	if len(got.Unavailable) != 1 || got.Unavailable[0].Capability != "goncho_reasoning_queue" {
		t.Fatalf("Unavailable = %+v, want queue-unavailable evidence", got.Unavailable)
	}

	ctx, err := svc.Context(context.Background(), ContextParams{
		Peer:       "telegram:6586915095",
		SessionKey: "session-import-public",
		MaxTokens:  400,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ctx.RecentMessages) != 1 || ctx.RecentMessages[0].Content != got.Messages[0].Content {
		t.Fatalf("RecentMessages = %+v, want imported chunk as normal session message", ctx.RecentMessages)
	}
}

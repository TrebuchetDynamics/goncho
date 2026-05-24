package goncho

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/goncho/memory"
)

func TestLocalE2E_ServiceLifecycleBuildsContextFromPublicAPIs(t *testing.T) {
	ctx := context.Background()
	store, err := memory.OpenSqlite(t.TempDir()+"/memory.db", 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer func() {
		if err := store.Close(ctx); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}()

	if err := RunMigrations(store.DB()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	svc := NewService(store.DB(), Config{
		WorkspaceID:    "local-e2e-workspace",
		ObserverPeerID: "assistant",
		RecentMessages: 4,
	}, nil)

	peer := "telegram:6586915095"
	sessionKey := "local-e2e-session"
	profileFact := "Prefers deterministic local SQLite smoke tests."
	userMessage := "Please test Goncho locally end to end before deciding later work."
	assistantMessage := "I will use public APIs only and verify the context pack."
	conclusion := "Goncho local E2E smoke uses deterministic SQLite only."

	if err := svc.SetProfile(ctx, peer, []string{profileFact}); err != nil {
		t.Fatalf("SetProfile: %v", err)
	}

	created, err := svc.CreateMessages(ctx, CreateMessagesParams{
		SessionKey: sessionKey,
		Messages: []CreateMessage{
			{Peer: peer, Role: "user", Content: userMessage, CreatedAt: time.Unix(1700000100, 0).UTC()},
			{Peer: "assistant", Role: "assistant", Content: assistantMessage, CreatedAt: time.Unix(1700000101, 0).UTC()},
		},
	})
	if err != nil {
		t.Fatalf("CreateMessages: %v", err)
	}
	if len(created.Messages) != 2 {
		t.Fatalf("created messages len = %d, want 2", len(created.Messages))
	}
	if created.Messages[0].Sequence != 1 || created.Messages[1].Sequence != 2 {
		t.Fatalf("created message sequences = %d,%d; want 1,2", created.Messages[0].Sequence, created.Messages[1].Sequence)
	}

	if _, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       peer,
		Conclusion: conclusion,
		SessionKey: sessionKey,
	}); err != nil {
		t.Fatalf("Conclude: %v", err)
	}

	contextResult, err := svc.Context(ctx, ContextParams{
		Peer:       peer,
		Query:      "deterministic SQLite",
		MaxTokens:  1000,
		SessionKey: sessionKey,
	})
	if err != nil {
		t.Fatalf("Context: %v", err)
	}
	if contextResult.WorkspaceID != "local-e2e-workspace" || contextResult.Peer != peer || contextResult.ObserverPeerID != "assistant" || contextResult.SessionKey != sessionKey {
		t.Fatalf("context identity = %+v", contextResult)
	}
	if !containsString(contextResult.PeerCard, profileFact) {
		t.Fatalf("context peer card = %#v, want %q", contextResult.PeerCard, profileFact)
	}
	if !containsString(contextResult.Conclusions, conclusion) {
		t.Fatalf("context conclusions = %#v, want %q", contextResult.Conclusions, conclusion)
	}
	if !messageSlicesContain(contextResult.RecentMessages, userMessage) {
		t.Fatalf("context recent messages = %#v, want user message %q", contextResult.RecentMessages, userMessage)
	}
	if !strings.Contains(contextResult.Representation, peer) || !strings.Contains(contextResult.Representation, conclusion) {
		t.Fatalf("context representation = %q, want peer and conclusion", contextResult.Representation)
	}

	searchResult, err := svc.Search(ctx, SearchParams{
		Peer:       peer,
		Query:      "deterministic SQLite",
		MaxTokens:  1000,
		SessionKey: sessionKey,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if !searchHitsContainSourceContent(searchResult.Results, "conclusion", conclusion) {
		t.Fatalf("search results = %#v, want conclusion hit %q", searchResult.Results, conclusion)
	}

	chatResult, err := svc.Chat(ctx, peer, ChatParams{
		Query:     "How should I run deterministic SQLite checks?",
		SessionID: sessionKey,
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	for _, want := range []string{
		"Query: How should I run deterministic SQLite checks?",
		"Reasoning level: low",
		conclusion,
	} {
		if !strings.Contains(chatResult.Content, want) {
			t.Fatalf("chat content missing %q in %q", want, chatResult.Content)
		}
	}

	mustRoundTripJSON(t, contextResult)
	mustRoundTripJSON(t, chatResult)
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func messageSlicesContain(messages []MessageSlice, content string) bool {
	for _, message := range messages {
		if message.Content == content {
			return true
		}
	}
	return false
}

func searchHitsContainSourceContent(hits []SearchHit, source, content string) bool {
	for _, hit := range hits {
		if hit.Source == source && hit.Content == content {
			return true
		}
	}
	return false
}

func mustRoundTripJSON(t *testing.T, value any) {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("Marshal %T: %v", value, err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("Unmarshal %T JSON shape: %v\n%s", value, err, raw)
	}
	if len(decoded) == 0 {
		t.Fatalf("%T JSON decoded as empty object: %s", value, raw)
	}
}

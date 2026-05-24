package goncho

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestGonchoHonchoSDKCompatibility_SessionMessageSearch(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	harness := NewHonchoSDKCompatibilityHarness(svc)
	ctx := context.Background()
	created, err := harness.SeedSession(ctx, HonchoSDKSessionSeed{
		PeerID:    "telegram:6586915095",
		SessionID: "sdk-session",
		PeerCard:  []string{"Prefers exact evidence-first reports"},
		Conclusions: []string{
			"The user prefers exact evidence-first reports.",
		},
		Messages: []HonchoSDKMessageInput{
			{
				PeerID:    "telegram:6586915095",
				Role:      "user",
				Content:   "Please keep reports exact and evidence-first.",
				CreatedAt: time.Unix(1_800_000_001, 0).UTC(),
				Metadata:  map[string]any{"sdk": "python", "index": float64(1)},
			},
			{
				PeerID:    "assistant",
				Role:      "assistant",
				Content:   "I will keep evidence first.",
				CreatedAt: time.Unix(1_800_000_002, 0).UTC(),
				Metadata:  map[string]any{"sdk": "typescript", "index": float64(2)},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if created.Workspace.ID != "default" {
		t.Fatalf("workspace id = %q, want default", created.Workspace.ID)
	}
	if created.Peer.ID != "telegram:6586915095" {
		t.Fatalf("peer id = %q, want seeded peer", created.Peer.ID)
	}
	if created.Session.ID != "sdk-session" {
		t.Fatalf("session id = %q, want sdk-session", created.Session.ID)
	}
	if len(created.Messages) != 2 {
		t.Fatalf("messages len = %d, want 2", len(created.Messages))
	}
	if created.Messages[0].PeerID != "telegram:6586915095" || created.Messages[0].SessionID != "sdk-session" {
		t.Fatalf("message response = %+v, want Honcho-shaped peer/session ids", created.Messages[0])
	}
	if created.Messages[1].Sequence != 2 {
		t.Fatalf("second sequence = %d, want 2", created.Messages[1].Sequence)
	}
	assertHonchoSDKJSONFields(t, created.Messages[0], "peer_id", "session_id", "created_at", "metadata")

	search, err := harness.Search(ctx, HonchoSDKSearchRequest{
		PeerID:    "telegram:6586915095",
		SessionID: "sdk-session",
		Query:     "evidence-first",
	})
	if err != nil {
		t.Fatal(err)
	}
	if search.WorkspaceID != "default" || search.PeerID != "telegram:6586915095" {
		t.Fatalf("search scope = %+v, want default workspace and seeded peer", search)
	}
	if len(search.Results) == 0 {
		t.Fatalf("search results empty, want seeded conclusion or turn evidence: %+v", search)
	}
	if !containsSDKSearchContent(search.Results, "evidence-first") {
		t.Fatalf("search results = %+v, want evidence-first content", search.Results)
	}
}

func TestGonchoHonchoSDKCompatibility_ContextPreview(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	harness := NewHonchoSDKCompatibilityHarness(svc)
	ctx := context.Background()
	messages := make([]HonchoSDKMessageInput, 0, 60)
	for i := 1; i <= 60; i++ {
		messages = append(messages, HonchoSDKMessageInput{
			PeerID:    "telegram:6586915095",
			Role:      "user",
			Content:   "turn evidence-first preference note",
			CreatedAt: time.Unix(1_800_001_000+int64(i), 0).UTC(),
		})
	}
	if _, err := harness.SeedSession(ctx, HonchoSDKSessionSeed{
		PeerID:    "telegram:6586915095",
		SessionID: "sdk-context",
		PeerCard:  []string{"Prefers exact evidence-first reports"},
		Conclusions: []string{
			"The user prefers exact evidence-first reports.",
		},
		Messages: messages,
	}); err != nil {
		t.Fatal(err)
	}

	preview, err := harness.ContextPreview(ctx, HonchoSDKContextPreviewRequest{
		PeerID:    "telegram:6586915095",
		SessionID: "sdk-context",
		Query:     "evidence-first",
		Tokens:    120,
	})
	if err != nil {
		t.Fatal(err)
	}

	if preview.WorkspaceID != "default" || preview.PeerID != "telegram:6586915095" || preview.SessionID != "sdk-context" {
		t.Fatalf("context scope = %+v, want Honcho-shaped workspace/peer/session ids", preview)
	}
	if !slices.Equal(preview.PeerCard, []string{"Prefers exact evidence-first reports"}) {
		t.Fatalf("peer card = %+v, want seeded card", preview.PeerCard)
	}
	if !strings.Contains(preview.Representation, "evidence-first") {
		t.Fatalf("representation = %q, want evidence-first memory", preview.Representation)
	}
	if preview.Summary == nil {
		t.Fatalf("summary = nil, want context preview summary; unavailable=%+v", preview.Unsupported)
	}
	if preview.Summary.Type != "long" {
		t.Fatalf("summary type = %q, want long", preview.Summary.Type)
	}
	assertHonchoSDKJSONFields(t, preview, "workspace_id", "peer_id", "session_id", "peer_card", "representation", "summary")
}

func TestGonchoHonchoSDKCompatibility_UnsupportedFlowEvidence(t *testing.T) {
	evidence := UnsupportedHonchoSDKFlow("POST", "/v3/workspaces/{workspace_id}/sessions/{session_id}/messages/bulk", "attachments", "stream")

	if evidence.Code != "sdk_flow_unsupported" {
		t.Fatalf("code = %q, want sdk_flow_unsupported", evidence.Code)
	}
	if evidence.Method != "POST" || evidence.Endpoint == "" {
		t.Fatalf("method/endpoint = %+v, want structured endpoint evidence", evidence)
	}
	if !slices.Equal(evidence.Fields, []string{"attachments", "stream"}) {
		t.Fatalf("fields = %+v, want unsupported fields", evidence.Fields)
	}
	raw, err := json.Marshal(evidence)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"code":"sdk_flow_unsupported"`, `"method":"POST"`, `"endpoint":`, `"fields":["attachments","stream"]`} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("unsupported evidence JSON = %s, want %s", raw, want)
		}
	}
}

func assertHonchoSDKJSONFields(t *testing.T, value any, fields ...string) {
	t.Helper()

	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, field := range fields {
		if _, ok := decoded[field]; !ok {
			t.Fatalf("JSON %s missing field %q", raw, field)
		}
	}
}

func containsSDKSearchContent(results []HonchoSDKSearchHit, needle string) bool {
	for _, result := range results {
		if strings.Contains(result.Content, needle) {
			return true
		}
	}
	return false
}

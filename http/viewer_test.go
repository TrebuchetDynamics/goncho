package gonchohttp

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/goncho/memory"
	goncho "github.com/TrebuchetDynamics/goncho/service"
)

func TestViewerSessionTimelineEndpointCombinesMessagesObservationsAndSummaries(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "timeline.db")
	store, err := memory.OpenSqlite(dbPath, 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(ctx)
	if err := goncho.RunMigrations(store.DB()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	workspace := "timeline-workspace"
	peer := "user-timeline"
	session := "timeline-session"
	svc := goncho.NewService(store.DB(), goncho.Config{WorkspaceID: workspace, ObserverPeerID: "assistant"}, nil)
	handler := NewServiceHandler(svc)

	messages, err := svc.CreateMessages(ctx, goncho.CreateMessagesParams{SessionKey: session, Messages: []goncho.CreateMessage{
		{Peer: peer, Role: "user", Content: "first timeline message"},
		{Peer: "assistant", Role: "assistant", Content: "assistant timeline response"},
	}})
	if err != nil {
		t.Fatalf("CreateMessages: %v", err)
	}
	if _, err := svc.Observe(ctx, goncho.ObservationParams{Kind: goncho.ObservationKindToolCall, PeerID: "assistant", SessionKey: session, Input: "bash go test", Success: boolPtr(true)}); err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if _, err := svc.CaptureHostHook(ctx, goncho.HostHookEvent{Event: goncho.HostHookSessionEnd, PeerID: "assistant", SessionKey: session, Summary: "timeline summary from host"}); err != nil {
		t.Fatalf("CaptureHostHook session end: %v", err)
	}
	if _, err := svc.CreateMessages(ctx, goncho.CreateMessagesParams{SessionKey: "other-session", Messages: []goncho.CreateMessage{{Peer: peer, Role: "user", Content: "do not include"}}}); err != nil {
		t.Fatalf("CreateMessages other session: %v", err)
	}

	timeline := getJSON[goncho.ViewerSessionTimeline](t, handler, "/v3/workspaces/"+workspace+"/viewer/sessions/"+session+"/timeline", http.StatusOK)
	if timeline.Status != "ok" || !timeline.ReadOnly || timeline.WorkspaceID != workspace || timeline.SessionKey != session {
		t.Fatalf("timeline header = %+v, want ok read-only workspace/session", timeline)
	}
	if len(timeline.Messages) != 2 || timeline.Messages[0].ID != messages.Messages[0].ID || timeline.Messages[1].ID != messages.Messages[1].ID {
		t.Fatalf("timeline messages = %+v, want seeded session messages only", timeline.Messages)
	}
	if len(timeline.Observations) != 2 {
		t.Fatalf("timeline observations len = %d, want tool_call plus session_end: %+v", len(timeline.Observations), timeline.Observations)
	}
	if len(timeline.Summaries) != 1 || timeline.Summaries[0].Content != "timeline summary from host" {
		t.Fatalf("timeline summaries = %+v, want host summary", timeline.Summaries)
	}
	for _, want := range []string{"message", "observation", "summary"} {
		if !timelineEventsContainType(timeline.Events, want) {
			t.Fatalf("timeline events = %+v, missing type %q", timeline.Events, want)
		}
	}
	if timelineEventsContainContent(timeline.Events, "do not include") {
		t.Fatalf("timeline events leaked another session: %+v", timeline.Events)
	}
	_ = requestJSON[map[string]any](t, handler, http.MethodPost, "/v3/workspaces/"+workspace+"/viewer/sessions/"+session+"/timeline", map[string]any{"mutate": true}, http.StatusNotFound)
}

func boolPtr(value bool) *bool {
	return &value
}

func timelineEventsContainType(events []goncho.ViewerTimelineEvent, want string) bool {
	for _, event := range events {
		if event.Type == want {
			return true
		}
	}
	return false
}

func timelineEventsContainContent(events []goncho.ViewerTimelineEvent, want string) bool {
	for _, event := range events {
		if event.Content == want {
			return true
		}
	}
	return false
}

func TestViewerEndpointReturnsReadOnlyWorkspaceSnapshot(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "viewer.db")
	store, err := memory.OpenSqlite(dbPath, 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(ctx)
	if err := goncho.RunMigrations(store.DB()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	workspace := "viewer-workspace"
	peer := "user-viewer"
	session := "viewer-session"
	svc := goncho.NewService(store.DB(), goncho.Config{WorkspaceID: workspace, ObserverPeerID: "assistant"}, nil)
	handler := NewServiceHandler(svc)

	if _, err := svc.CreateMessages(ctx, goncho.CreateMessagesParams{SessionKey: session, Messages: []goncho.CreateMessage{{Peer: peer, Role: "user", Content: "show me viewer state"}}}); err != nil {
		t.Fatalf("CreateMessages: %v", err)
	}
	if _, err := svc.Conclude(ctx, goncho.ConcludeParams{Peer: peer, SessionKey: session, Conclusion: "Viewer endpoint is read-only JSON."}); err != nil {
		t.Fatalf("Conclude: %v", err)
	}
	if _, err := svc.Observe(ctx, goncho.ObservationParams{Kind: goncho.ObservationKindUserPrompt, PeerID: peer, SessionKey: session, Input: "show me viewer state"}); err != nil {
		t.Fatalf("Observe: %v", err)
	}
	review, err := svc.CreateReviewItem(ctx, goncho.ReviewItemCreateParams{Kind: goncho.ReviewKindStale, PeerID: peer, SessionKey: session, SubjectID: "viewer-memory", Reason: "viewer test review item"})
	if err != nil {
		t.Fatalf("CreateReviewItem: %v", err)
	}
	otherSvc := goncho.NewService(store.DB(), goncho.Config{WorkspaceID: "other-workspace", ObserverPeerID: "assistant"}, nil)
	if _, err := otherSvc.CreateMessages(ctx, goncho.CreateMessagesParams{SessionKey: "other-session", Messages: []goncho.CreateMessage{{Peer: "other-peer", Role: "user", Content: "other workspace"}}}); err != nil {
		t.Fatalf("CreateMessages other workspace: %v", err)
	}

	snapshot := getJSON[goncho.ViewerSnapshot](t, handler, "/v3/workspaces/"+workspace+"/viewer", http.StatusOK)
	if snapshot.Status != "ok" || !snapshot.ReadOnly {
		t.Fatalf("snapshot status/read_only = %q/%v, want ok/read-only", snapshot.Status, snapshot.ReadOnly)
	}
	if snapshot.WorkspaceID != workspace || snapshot.ObserverPeerID != "assistant" {
		t.Fatalf("snapshot identity = %+v, want workspace/observer", snapshot)
	}
	if snapshot.DB.Path != dbPath {
		t.Fatalf("db path = %q, want %q", snapshot.DB.Path, dbPath)
	}
	if snapshot.Counts.Sessions != 1 || snapshot.Counts.Messages != 1 || snapshot.Counts.Observations != 1 || snapshot.Counts.Conclusions != 1 || snapshot.Counts.ReviewOpen != 1 {
		t.Fatalf("counts = %+v, want one session/message/observation/conclusion/open review", snapshot.Counts)
	}
	if len(snapshot.LatestObservations) != 1 || snapshot.LatestObservations[0].Kind != goncho.ObservationKindUserPrompt {
		t.Fatalf("latest observations = %+v, want user_prompt", snapshot.LatestObservations)
	}
	if len(snapshot.LatestConclusions) != 1 || snapshot.LatestConclusions[0].Content != "Viewer endpoint is read-only JSON." {
		t.Fatalf("latest conclusions = %+v, want seeded conclusion", snapshot.LatestConclusions)
	}
	if snapshot.ReviewQueue.Open != 1 || len(snapshot.ReviewQueue.LatestOpen) != 1 || snapshot.ReviewQueue.LatestOpen[0].ID != review.ID {
		t.Fatalf("review queue = %+v, want seeded open review", snapshot.ReviewQueue)
	}

	_ = requestJSON[map[string]any](t, handler, http.MethodPost, "/v3/workspaces/"+workspace+"/viewer", map[string]any{"mutate": true}, http.StatusNotFound)
}

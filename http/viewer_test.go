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

func TestViewerRecallTraceEndpointReturnsSelectedAndRejectedCandidates(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "recall-viewer.db")
	store, err := memory.OpenSqlite(dbPath, 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(ctx)
	if err := goncho.RunMigrations(store.DB()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	workspace := "recall-viewer-workspace"
	peer := "user-recall"
	session := "recall-viewer-session"
	svc := goncho.NewService(store.DB(), goncho.Config{WorkspaceID: workspace, ObserverPeerID: "assistant"}, nil)
	handler := NewServiceHandler(svc)

	// Seed a conclusion that recall can find.
	if _, err := svc.Conclude(ctx, goncho.ConcludeParams{Peer: peer, SessionKey: session, Conclusion: "The recall viewer endpoint returns trace JSON."}); err != nil {
		t.Fatalf("Conclude: %v", err)
	}

	recallResult := getJSON[goncho.ViewerRecallTrace](t, handler,
		"/v3/workspaces/"+workspace+"/viewer/recall?peer="+peer+"&query=recall+viewer+endpoint",
		http.StatusOK)

	if recallResult.Status != "ok" || !recallResult.ReadOnly {
		t.Fatalf("recall viewer status/read_only = %q/%v, want ok/read-only", recallResult.Status, recallResult.ReadOnly)
	}
	if recallResult.WorkspaceID != workspace || recallResult.Peer != peer || recallResult.Query != "recall viewer endpoint" {
		t.Fatalf("recall viewer identity = %+v, want workspace/peer/query", recallResult)
	}
	if recallResult.Trace.TraceID == "" {
		t.Fatalf("recall viewer trace missing trace_id")
	}
	if len(recallResult.Trace.Selected) == 0 {
		t.Fatalf("recall viewer trace has zero selected candidates, want at least one from seeded conclusion")
	}
	// Verify rejected candidates are included even when empty.
	if recallResult.Trace.Rejected == nil {
		t.Fatalf("recall viewer trace rejected is nil, want empty slice")
	}

	// Verify warnings are included even when empty.
	if recallResult.Trace.Warnings == nil {
		t.Fatalf("recall viewer trace warnings is nil, want empty slice")
	}

	// Verify the seeded conclusion appears in selected candidates.
	found := false
	for _, s := range recallResult.Trace.Selected {
		if s.Candidate.Content == "The recall viewer endpoint returns trace JSON." {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("recall viewer trace selected = %+v, missing seeded conclusion", recallResult.Trace.Selected)
	}

	// Verify scoring config is present.
	if recallResult.Trace.ScoringConfig.Version == "" {
		t.Fatalf("recall viewer trace missing scoring config version")
	}

	// POST should return 404 (read-only viewer).
	_ = requestJSON[map[string]any](t, handler, http.MethodPost,
		"/v3/workspaces/"+workspace+"/viewer/recall",
		map[string]any{"mutate": true},
		http.StatusNotFound)
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
	failed := false
	for _, id := range []string{"viewer-fail-1", "viewer-fail-2"} {
		if _, err := svc.Observe(ctx, goncho.ObservationParams{ID: id, Kind: goncho.ObservationKindToolError, PeerID: peer, SessionKey: session, Success: &failed, Input: "private viewer command", Output: "private viewer stack", Metadata: map[string]string{"tool_name": "bash"}}); err != nil {
			t.Fatalf("Observe failure %s: %v", id, err)
		}
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
	if snapshot.Counts.Sessions != 1 || snapshot.Counts.Messages != 1 || snapshot.Counts.Observations != 3 || snapshot.Counts.Conclusions != 1 || snapshot.Counts.ReviewOpen != 1 {
		t.Fatalf("counts = %+v, want one session/message/conclusion/open review and three observations", snapshot.Counts)
	}
	if len(snapshot.NegativeEvidenceCandidates) != 1 || snapshot.NegativeEvidenceCandidates[0].FailureCount != 2 || snapshot.NegativeEvidenceCandidates[0].ToolName != "bash" {
		t.Fatalf("negative evidence candidates = %+v, want repeated bash failure", snapshot.NegativeEvidenceCandidates)
	}
	if snapshot.NegativeEvidenceCandidates[0].String() == "" || timelineEventsContainContent([]goncho.ViewerTimelineEvent{{Content: snapshot.NegativeEvidenceCandidates[0].String()}}, "private viewer command") {
		t.Fatalf("negative evidence candidate leaked raw content: %+v", snapshot.NegativeEvidenceCandidates[0])
	}
	if len(snapshot.LatestObservations) != 3 || snapshot.LatestObservations[0].Kind != goncho.ObservationKindToolError {
		t.Fatalf("latest observations = %+v, want latest tool_error first", snapshot.LatestObservations)
	}
	if len(snapshot.LatestConclusions) != 1 || snapshot.LatestConclusions[0].Content != "Viewer endpoint is read-only JSON." {
		t.Fatalf("latest conclusions = %+v, want seeded conclusion", snapshot.LatestConclusions)
	}
	if snapshot.ReviewQueue.Open != 1 || len(snapshot.ReviewQueue.LatestOpen) != 1 || snapshot.ReviewQueue.LatestOpen[0].ID != review.ID {
		t.Fatalf("review queue = %+v, want seeded open review", snapshot.ReviewQueue)
	}

	_ = requestJSON[map[string]any](t, handler, http.MethodPost, "/v3/workspaces/"+workspace+"/viewer", map[string]any{"mutate": true}, http.StatusNotFound)
}

package goncho

import (
	"context"
	"testing"
)

func TestGonchoGoalMetaanalysisPublicToolSurfaceWorksEndToEnd(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	peer := "peer-tools"
	sessionKey := "session-tools"

	rememberTool := NewGonchoRememberTool(svc)
	searchTool := NewGonchoSearchTool(svc)
	contextTool := NewGonchoContextTool(svc)
	reviewTool := NewReviewTool(svc)
	handoffTool := NewGonchoHandoffTool(newMockToolStore())

	for _, tc := range []struct {
		name string
		tool interface{ Name() string }
	}{
		{"goncho_remember", rememberTool},
		{"goncho_search", searchTool},
		{"goncho_context", contextTool},
		{"goncho_review", reviewTool},
		{"goncho_handoff", handoffTool},
	} {
		if tc.tool.Name() != tc.name {
			t.Fatalf("tool name = %q, want %q", tc.tool.Name(), tc.name)
		}
	}

	remembered := executeMemoryTool(t, ctx, rememberTool, `{"peer_id":"`+peer+`","content":"Goncho public tool surface remembers local-first claims.","session_key":"`+sessionKey+`"}`)
	if stringField(t, remembered, "action") != "remember" || stringField(t, remembered, "status") == "" {
		t.Fatalf("remember output = %+v", remembered)
	}

	searched := executeMemoryTool(t, ctx, searchTool, `{"peer_id":"`+peer+`","query":"local-first claims","session_key":"`+sessionKey+`"}`)
	if intField(t, searched, "count") != 1 {
		t.Fatalf("search output = %+v, want one result", searched)
	}

	contexted := executeMemoryTool(t, ctx, contextTool, `{"peer_id":"`+peer+`","query":"local-first claims","session_key":"`+sessionKey+`"}`)
	if stringField(t, contexted, "peer") != peer || stringField(t, contexted, "representation") == "" {
		t.Fatalf("context output = %+v, want peer and representation", contexted)
	}

	item, err := svc.CreateReviewItem(ctx, ReviewItemCreateParams{Kind: ReviewKindStale, PeerID: peer, SubjectID: "memory-stale", Reason: "public tool surface review"})
	if err != nil {
		t.Fatalf("CreateReviewItem: %v", err)
	}
	listed := executeMemoryTool(t, ctx, reviewTool, `{"action":"list","peer_id":"`+peer+`","status":"open"}`)
	if intField(t, listed, "count") != 1 {
		t.Fatalf("review list output = %+v, want one open item", listed)
	}
	resolved := executeMemoryTool(t, ctx, reviewTool, `{"action":"resolve","id":"`+item.ID+`","resolution":"verified","resolved_by":"agent:mineru","resolution_reason":"public surface checked"}`)
	if stringField(t, resolved, "status") != string(ReviewStatusResolved) {
		t.Fatalf("review resolve output = %+v", resolved)
	}

	saved := executeMemoryTool(t, ctx, handoffTool, `{"action":"save","session_id":"`+sessionKey+`","content":"Next agent should run go test ./... before claiming Goncho done."}`)
	if stringField(t, saved, "action") != "save" || stringField(t, saved, "id") == "" {
		t.Fatalf("handoff save output = %+v", saved)
	}
	loaded := executeMemoryTool(t, ctx, handoffTool, `{"action":"load","session_id":"`+sessionKey+`"}`)
	if intField(t, loaded, "count") != 1 {
		t.Fatalf("handoff load output = %+v, want one handoff", loaded)
	}
}

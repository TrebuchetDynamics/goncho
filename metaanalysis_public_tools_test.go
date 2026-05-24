package goncho

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestGonchoGoalPublicContextToolGeneratesPrimerWithinTokenBudgetE2E(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	peer := "peer-primer-budget"
	sessionKey := "session-primer-budget"

	if err := svc.SetProfile(ctx, peer, []string{"Prefers compact token-budgeted primers"}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       peer,
		Conclusion: "Generated primers should keep local-first recall compact under max_tokens.",
		SessionKey: sessionKey,
	}); err != nil {
		t.Fatal(err)
	}
	seedSummaryContextTurns(t, ctx, svc, sessionKey, 3, 2)

	contextTool := NewGonchoContextTool(svc)
	primer := executeMemoryTool(t, ctx, contextTool, `{"peer_id":"`+peer+`","query":"token-budgeted primer","session_key":"`+sessionKey+`","max_tokens":4}`)

	if representation := stringField(t, primer, "representation"); !strings.Contains(representation, "Representation for "+peer) {
		t.Fatalf("representation = %q, want generated primer for peer", representation)
	}
	recent, ok := primer["recent_messages"].([]any)
	if !ok {
		t.Fatalf("recent_messages = %#v, want JSON array", primer["recent_messages"])
	}
	if len(recent) != 2 {
		t.Fatalf("recent_messages len = %d, want 2 messages inside 4-token budget", len(recent))
	}
	raw, err := json.Marshal(primer)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, want := range []string{"turn-02 word", "turn-03 word"} {
		if !strings.Contains(text, want) {
			t.Fatalf("primer output missing %q: %s", want, text)
		}
	}
	if strings.Contains(text, "turn-01 word") {
		t.Fatalf("primer output included oldest message outside token budget: %s", text)
	}
}

func TestGonchoRecallToolCompactOutputKeepsDiagnosticsWithoutLargeTracePayload(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	peer := "peer-recall-compact"
	sessionKey := "session-recall-compact"

	if _, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       peer,
		Conclusion: "Compact recall output should keep diagnostics without full trace payloads.",
		SessionKey: sessionKey,
	}); err != nil {
		t.Fatal(err)
	}

	recalled := executeMemoryTool(t, ctx, NewGonchoRecallTool(svc), `{"peer_id":"`+peer+`","query":"compact diagnostics","session_key":"`+sessionKey+`","limit":3,"compact":true}`)
	if stringField(t, recalled, "action") != "recall" || intField(t, recalled, "selected_count") != 1 {
		t.Fatalf("recall output = %+v, want one compact selected result", recalled)
	}
	if stringField(t, recalled, "trace_id") == "" || stringField(t, recalled, "replay_contract") != "deterministic_replay_from_recall_trace" {
		t.Fatalf("recall output = %+v, want trace identity and replay contract", recalled)
	}
	if _, ok := recalled["diagnostics"].(map[string]any); !ok {
		t.Fatalf("diagnostics = %#v, want compact diagnostics object", recalled["diagnostics"])
	}
	for _, omitted := range []string{"trace", "replay", "selected", "warnings", "diagnostics_text"} {
		if _, ok := recalled[omitted]; ok {
			t.Fatalf("compact recall output included %q: %+v", omitted, recalled)
		}
	}
}

func TestGonchoGoalMetaanalysisPublicToolSurfaceWorksEndToEnd(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	peer := "peer-tools"
	sessionKey := "session-tools"

	rememberTool := NewGonchoRememberTool(svc)
	searchTool := NewGonchoSearchTool(svc)
	recallTool := NewGonchoRecallTool(svc)
	contextTool := NewGonchoContextTool(svc)
	reviewTool := NewReviewTool(svc)
	handoffTool := NewGonchoHandoffTool(newMockToolStore())

	for _, tc := range []struct {
		name string
		tool interface{ Name() string }
	}{
		{"goncho_remember", rememberTool},
		{"goncho_search", searchTool},
		{"goncho_recall", recallTool},
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

	recalled := executeMemoryTool(t, ctx, recallTool, `{"peer_id":"`+peer+`","query":"local-first claims","session_key":"`+sessionKey+`","limit":3}`)
	if stringField(t, recalled, "action") != "recall" || intField(t, recalled, "selected_count") != 1 {
		t.Fatalf("recall output = %+v, want one selected trace result", recalled)
	}
	if stringField(t, recalled, "trace_id") == "" || stringField(t, recalled, "replay_contract") != "deterministic_replay_from_recall_trace" {
		t.Fatalf("recall output = %+v, want trace and replay contract", recalled)
	}
	diagnostics, ok := recalled["diagnostics"].(map[string]any)
	if !ok {
		t.Fatalf("recall diagnostics = %#v, want JSON object", recalled["diagnostics"])
	}
	if stringField(t, diagnostics, "projection_invariant") != "no_projection_without_recall_trace" || intField(t, diagnostics, "selected_count") != 1 {
		t.Fatalf("diagnostics = %+v, want projection invariant and one selected item", diagnostics)
	}
	if text := stringField(t, recalled, "diagnostics_text"); !strings.Contains(text, "Goncho recall diagnostics") || !strings.Contains(text, "selected candidates") {
		t.Fatalf("diagnostics_text = %q, want formatted recall diagnostics", text)
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

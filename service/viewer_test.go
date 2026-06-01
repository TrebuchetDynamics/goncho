package goncho

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/goncho/memory"
)

func TestRecallViewerPayloadIncludesRejectedAndWarnings(t *testing.T) {
	ctx := context.Background()
	store, err := memory.OpenSqlite(filepath.Join(t.TempDir(), "viewer-recall.db"), 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(ctx)
	if err := RunMigrations(store.DB()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	svc := NewService(store.DB(), Config{WorkspaceID: "viewer-recall-workspace", ObserverPeerID: "assistant"}, nil)
	for _, conclusion := range []string{
		"Recall viewer selected evidence about amber adapters.",
		"Recall viewer rejected evidence about amber adapters.",
	} {
		if _, err := svc.Conclude(ctx, ConcludeParams{Peer: "viewer-peer", SessionKey: "viewer-session", Conclusion: conclusion}); err != nil {
			t.Fatalf("Conclude %q: %v", conclusion, err)
		}
	}
	if _, err := svc.Conclude(ctx, ConcludeParams{Peer: "viewer-peer", SessionKey: "other-session", Conclusion: "Other session amber adapter evidence should stay filtered."}); err != nil {
		t.Fatalf("Conclude other session: %v", err)
	}

	payload, err := svc.ViewerRecallTrace(ctx, "viewer-peer", "amber adapters", "viewer-session", 1)
	if err != nil {
		t.Fatalf("ViewerRecallTrace: %v", err)
	}
	if payload.Status != "ok" || !payload.ReadOnly || payload.SessionKey != "viewer-session" {
		t.Fatalf("viewer payload header = %+v, want ok read-only session-scoped", payload)
	}
	if len(payload.Trace.Selected) != 1 {
		t.Fatalf("selected = %+v, want one selected with limit=1", payload.Trace.Selected)
	}
	if len(payload.Trace.Rejected) == 0 {
		t.Fatalf("rejected = %+v, want non-selected in-session candidate", payload.Trace.Rejected)
	}
	if payload.Trace.Warnings == nil {
		t.Fatalf("warnings is nil, want empty warning slice for viewer clients")
	}
	if payload.Diagnostics.TraceID != payload.Trace.TraceID || payload.Diagnostics.SelectedCount != len(payload.Trace.Selected) || payload.Diagnostics.RejectedCount != len(payload.Trace.Rejected) {
		t.Fatalf("diagnostics = %+v trace = %+v, want trace-aligned counts", payload.Diagnostics, payload.Trace)
	}
	if payload.DiagnosticsText == "" {
		t.Fatalf("diagnostics_text is empty")
	}
	for _, selected := range payload.Trace.Selected {
		if selected.Candidate.SessionID != "viewer-session" {
			t.Fatalf("selected leaked other session: %+v", selected)
		}
	}
	for _, rejected := range payload.Trace.Rejected {
		if rejected.Candidate.SessionID != "viewer-session" {
			t.Fatalf("rejected leaked other session: %+v", rejected)
		}
	}
}

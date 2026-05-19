package goncho

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver"
)

func newTestProposalsDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`
		CREATE TABLE memory_proposals (
			id               INTEGER PRIMARY KEY AUTOINCREMENT,
			proposal_id      TEXT    NOT NULL UNIQUE,
			subtask_id       TEXT    NOT NULL,
			child_agent_id   TEXT    NOT NULL,
			parent_agent_id  TEXT    NOT NULL,
			proposed_tier    TEXT    NOT NULL CHECK(proposed_tier IN ('global','project','task','workspace','decision')),
			kind             TEXT    NOT NULL CHECK(kind IN ('fact','preference','decision','observation','report','artifact')),
			content          TEXT    NOT NULL,
			evidence_json    TEXT    NOT NULL DEFAULT '{}',
			status           TEXT    NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','accepted','rejected')),
			reviewed_by      TEXT,
			reviewed_at      INTEGER,
			committed_memory_id TEXT,
			created_at       INTEGER NOT NULL
		);
	`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestSubmitProposal_ValidInput(t *testing.T) {
	db := newTestProposalsDB(t)
	ctx := context.Background()

	ref, err := SubmitProposal(ctx, db, "sub_1", "child_a", "parent_a", "test content", TierProject, KindFact, map[string]string{"source": "tool"})
	if err != nil {
		t.Fatalf("SubmitProposal: %v", err)
	}
	if ref.ProposalID == "" {
		t.Error("expected non-empty ProposalID")
	}
	if ref.Status != StatusPending {
		t.Errorf("expected status pending, got %s", ref.Status)
	}

	p, err := GetProposal(ctx, db, ref.ProposalID)
	if err != nil {
		t.Fatalf("GetProposal: %v", err)
	}
	if p.Content != "test content" {
		t.Errorf("content = %q, want %q", p.Content, "test content")
	}
	if p.ProposedTier != TierProject {
		t.Errorf("tier = %q, want %q", p.ProposedTier, TierProject)
	}
	if p.Kind != KindFact {
		t.Errorf("kind = %q, want %q", p.Kind, KindFact)
	}
}

func TestSubmitProposal_InvalidTier(t *testing.T) {
	db := newTestProposalsDB(t)
	ctx := context.Background()

	_, err := SubmitProposal(ctx, db, "sub_1", "child_a", "parent_a", "content", MemoryTier("bogus"), KindFact, nil)
	if err == nil {
		t.Fatal("expected error for invalid tier")
	}
}

func TestSubmitProposal_InvalidKind(t *testing.T) {
	db := newTestProposalsDB(t)
	ctx := context.Background()

	_, err := SubmitProposal(ctx, db, "sub_1", "child_a", "parent_a", "content", TierProject, ProposalKind("bogus"), nil)
	if err == nil {
		t.Fatal("expected error for invalid kind")
	}
}

func TestListPendingProposals(t *testing.T) {
	db := newTestProposalsDB(t)
	ctx := context.Background()

	_, _ = SubmitProposal(ctx, db, "sub_1", "child_a", "parent_a", "first", TierTask, KindObservation, nil)
	_, _ = SubmitProposal(ctx, db, "sub_2", "child_b", "parent_a", "second", TierProject, KindFact, nil)
	_, _ = SubmitProposal(ctx, db, "sub_3", "child_c", "other_parent", "third", TierGlobal, KindReport, nil)

	pending, err := ListPendingProposals(ctx, db, "parent_a")
	if err != nil {
		t.Fatalf("ListPendingProposals: %v", err)
	}
	if len(pending) != 2 {
		t.Fatalf("expected 2 pending for parent_a, got %d", len(pending))
	}
	contents := map[string]bool{}
	for _, p := range pending {
		contents[p.Content] = true
	}
	if !contents["first"] || !contents["second"] {
		t.Errorf("expected proposals {first, second}, got %v", contents)
	}
}

func TestAcceptProposal(t *testing.T) {
	db := newTestProposalsDB(t)
	ctx := context.Background()

	ref, _ := SubmitProposal(ctx, db, "sub_1", "child_a", "parent_a", "content", TierProject, KindFact, nil)

	memID := "mem_committed_123"
	err := AcceptProposal(ctx, db, ref.ProposalID, "parent_a", &memID)
	if err != nil {
		t.Fatalf("AcceptProposal: %v", err)
	}

	p, _ := GetProposal(ctx, db, ref.ProposalID)
	if p.Status != StatusAccepted {
		t.Errorf("status = %q, want %q", p.Status, StatusAccepted)
	}
	if p.CommittedMemoryID == nil || *p.CommittedMemoryID != memID {
		t.Errorf("committed_memory_id = %v, want %q", p.CommittedMemoryID, memID)
	}
	if p.ReviewedBy == nil || *p.ReviewedBy != "parent_a" {
		t.Errorf("reviewed_by = %v, want %q", p.ReviewedBy, "parent_a")
	}
}

func TestAcceptProposal_NilMemoryID(t *testing.T) {
	db := newTestProposalsDB(t)
	ctx := context.Background()

	ref, _ := SubmitProposal(ctx, db, "sub_1", "child_a", "parent_a", "content", TierProject, KindDecision, nil)

	err := AcceptProposal(ctx, db, ref.ProposalID, "parent_a", nil)
	if err != nil {
		t.Fatalf("AcceptProposal: %v", err)
	}

	p, _ := GetProposal(ctx, db, ref.ProposalID)
	if p.Status != StatusAccepted {
		t.Errorf("status = %q, want %q", p.Status, StatusAccepted)
	}
	if p.CommittedMemoryID != nil {
		t.Errorf("committed_memory_id should be nil, got %v", p.CommittedMemoryID)
	}
}

func TestRejectProposal(t *testing.T) {
	db := newTestProposalsDB(t)
	ctx := context.Background()

	ref, _ := SubmitProposal(ctx, db, "sub_1", "child_a", "parent_a", "content", TierTask, KindObservation, nil)

	err := RejectProposal(ctx, db, ref.ProposalID, "parent_a")
	if err != nil {
		t.Fatalf("RejectProposal: %v", err)
	}

	p, _ := GetProposal(ctx, db, ref.ProposalID)
	if p.Status != StatusRejected {
		t.Errorf("status = %q, want %q", p.Status, StatusRejected)
	}
}

func TestAcceptProposal_AlreadyAccepted(t *testing.T) {
	db := newTestProposalsDB(t)
	ctx := context.Background()

	ref, _ := SubmitProposal(ctx, db, "sub_1", "child_a", "parent_a", "content", TierProject, KindFact, nil)
	_ = AcceptProposal(ctx, db, ref.ProposalID, "parent_a", nil)

	err := AcceptProposal(ctx, db, ref.ProposalID, "parent_a", nil)
	if err == nil {
		t.Fatal("expected error accepting already-accepted proposal")
	}
}

func TestRejectProposal_AlreadyRejected(t *testing.T) {
	db := newTestProposalsDB(t)
	ctx := context.Background()

	ref, _ := SubmitProposal(ctx, db, "sub_1", "child_a", "parent_a", "content", TierProject, KindFact, nil)
	_ = RejectProposal(ctx, db, ref.ProposalID, "parent_a")

	err := RejectProposal(ctx, db, ref.ProposalID, "parent_a")
	if err == nil {
		t.Fatal("expected error rejecting already-rejected proposal")
	}
}

func TestGetProposal_NotFound(t *testing.T) {
	db := newTestProposalsDB(t)
	ctx := context.Background()

	_, err := GetProposal(ctx, db, "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent proposal")
	}
}

func TestProposalEvidenceRoundTrip(t *testing.T) {
	db := newTestProposalsDB(t)
	ctx := context.Background()

	evidence := map[string]any{
		"source":     "tool_call",
		"tool_name":  "Read",
		"confidence": 0.95,
	}
	ref, err := SubmitProposal(ctx, db, "sub_1", "child_a", "parent_a", "evidence test", TierDecision, KindFact, evidence)
	if err != nil {
		t.Fatalf("SubmitProposal: %v", err)
	}

	p, err := GetProposal(ctx, db, ref.ProposalID)
	if err != nil {
		t.Fatalf("GetProposal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(p.EvidenceJSON, &decoded); err != nil {
		t.Fatalf("unmarshal evidence: %v", err)
	}
	if decoded["source"] != "tool_call" {
		t.Errorf("evidence.source = %v, want %q", decoded["source"], "tool_call")
	}
}

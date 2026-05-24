package skillproposals

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
)

func TestSkillLearningProposal_SubmitAndEvidenceRoundTrip(t *testing.T) {
	db := migratedSkillProposalTestDB(t)
	ctx := context.Background()

	createdAt := time.Date(2026, 5, 20, 15, 0, 0, 0, time.UTC)
	ref, err := SubmitSkillLearningProposal(ctx, db, SkillLearningProposalCreateParams{
		WorkspaceID: "workspace-a",
		SkillName:   "gormes-git",
		SourceTask:  "resolved repeated git blocker with build artifact evidence",
		DraftBody:   "# gormes-git\n\nCheck build artifacts before commit.",
		CreatedBy:   "agent:mineru",
		Evidence: map[string]any{
			"tool_calls": float64(7),
			"validation": "git diff --check",
			"outcome":    "success",
		},
		CreatedAt: createdAt,
	})
	if err != nil {
		t.Fatalf("SubmitSkillLearningProposal: %v", err)
	}
	if ref.ProposalID == "" || ref.Status != SkillLearningProposalPending {
		t.Fatalf("ref = %+v, want pending id", ref)
	}

	proposal, err := GetSkillLearningProposal(ctx, db, ref.ProposalID)
	if err != nil {
		t.Fatalf("GetSkillLearningProposal: %v", err)
	}
	if proposal.SkillName != "gormes-git" || proposal.Status != SkillLearningProposalPending {
		t.Fatalf("proposal = %+v", proposal)
	}
	var evidence map[string]any
	if err := json.Unmarshal(proposal.EvidenceJSON, &evidence); err != nil {
		t.Fatalf("evidence json: %v", err)
	}
	if evidence["validation"] != "git diff --check" || evidence["outcome"] != "success" {
		t.Fatalf("evidence = %#v", evidence)
	}
}

func TestSkillLearningProposal_InvalidInputFailsClosed(t *testing.T) {
	db := migratedSkillProposalTestDB(t)
	ctx := context.Background()

	cases := []SkillLearningProposalCreateParams{
		{SkillName: "", SourceTask: "task", DraftBody: "draft", CreatedBy: "agent:mineru"},
		{SkillName: "skill", SourceTask: "", DraftBody: "draft", CreatedBy: "agent:mineru"},
		{SkillName: "skill", SourceTask: "task", DraftBody: "", CreatedBy: "agent:mineru"},
		{SkillName: "skill", SourceTask: "task", DraftBody: "draft", CreatedBy: ""},
	}
	for _, tc := range cases {
		if _, err := SubmitSkillLearningProposal(ctx, db, tc); err == nil {
			t.Fatalf("expected invalid proposal to fail: %+v", tc)
		}
	}
}

func TestSkillLearningProposal_ListPendingScopedAndNewestFirst(t *testing.T) {
	db := migratedSkillProposalTestDB(t)
	ctx := context.Background()
	base := time.Date(2026, 5, 20, 16, 0, 0, 0, time.UTC)

	_, _ = SubmitSkillLearningProposal(ctx, db, SkillLearningProposalCreateParams{WorkspaceID: "workspace-a", SkillName: "old", SourceTask: "task old", DraftBody: "draft old", CreatedBy: "agent:mineru", CreatedAt: base})
	newer, _ := SubmitSkillLearningProposal(ctx, db, SkillLearningProposalCreateParams{WorkspaceID: "workspace-a", SkillName: "new", SourceTask: "task new", DraftBody: "draft new", CreatedBy: "agent:mineru", CreatedAt: base.Add(time.Minute)})
	_, _ = SubmitSkillLearningProposal(ctx, db, SkillLearningProposalCreateParams{WorkspaceID: "other-workspace", SkillName: "other", SourceTask: "task other", DraftBody: "draft other", CreatedBy: "agent:mineru", CreatedAt: base.Add(2 * time.Minute)})

	items, err := ListSkillLearningProposals(ctx, db, SkillLearningProposalQuery{WorkspaceID: "workspace-a", Status: SkillLearningProposalPending, Limit: 10})
	if err != nil {
		t.Fatalf("ListSkillLearningProposals: %v", err)
	}
	if len(items.Items) != 2 {
		t.Fatalf("pending count = %d, want 2: %+v", len(items.Items), items.Items)
	}
	if items.Items[0].ProposalID != newer.ProposalID || items.Items[0].SkillName != "new" {
		t.Fatalf("order = %+v, want newest first", items.Items)
	}
}

func TestSkillLearningProposal_ApproveRejectAndDuplicateReview(t *testing.T) {
	db := migratedSkillProposalTestDB(t)
	ctx := context.Background()

	approveRef, _ := SubmitSkillLearningProposal(ctx, db, SkillLearningProposalCreateParams{WorkspaceID: "workspace-a", SkillName: "approve-me", SourceTask: "task", DraftBody: "draft", CreatedBy: "agent:mineru"})
	approved, err := ApproveSkillLearningProposal(ctx, db, SkillLearningProposalReviewParams{ProposalID: approveRef.ProposalID, ReviewedBy: "human:juan", ReviewReason: "validated against tests"})
	if err != nil {
		t.Fatalf("ApproveSkillLearningProposal: %v", err)
	}
	if approved.Status != SkillLearningProposalApproved || approved.ReviewedBy != "human:juan" || approved.ReviewReason != "validated against tests" || approved.ReviewedAt == nil {
		t.Fatalf("approved = %+v", approved)
	}
	if _, err := RejectSkillLearningProposal(ctx, db, SkillLearningProposalReviewParams{ProposalID: approveRef.ProposalID, ReviewedBy: "human:juan", ReviewReason: "second review"}); err == nil {
		t.Fatal("expected duplicate review to fail")
	}

	rejectRef, _ := SubmitSkillLearningProposal(ctx, db, SkillLearningProposalCreateParams{WorkspaceID: "workspace-a", SkillName: "reject-me", SourceTask: "task", DraftBody: "draft", CreatedBy: "agent:mineru"})
	rejected, err := RejectSkillLearningProposal(ctx, db, SkillLearningProposalReviewParams{ProposalID: rejectRef.ProposalID, ReviewedBy: "human:juan", ReviewReason: "overfit to one invoice"})
	if err != nil {
		t.Fatalf("RejectSkillLearningProposal: %v", err)
	}
	if rejected.Status != SkillLearningProposalRejected || rejected.ReviewReason != "overfit to one invoice" {
		t.Fatalf("rejected = %+v", rejected)
	}
}

func migratedSkillProposalTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", t.TempDir()+"/skill_proposals.db")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := ensureSkillLearningProposalTable(context.Background(), db); err != nil {
		t.Fatalf("migrate skill proposals: %v", err)
	}
	return db
}

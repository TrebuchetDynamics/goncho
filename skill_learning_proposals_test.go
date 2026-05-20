package goncho

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestSkillLearningProposal_SubmitAndEvidenceRoundTrip(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	createdAt := time.Date(2026, 5, 20, 15, 0, 0, 0, time.UTC)
	ref, err := svc.SubmitSkillLearningProposal(ctx, SkillLearningProposalCreateParams{
		WorkspaceID: svc.workspaceID,
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

	proposal, err := svc.GetSkillLearningProposal(ctx, ref.ProposalID)
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
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	cases := []SkillLearningProposalCreateParams{
		{SkillName: "", SourceTask: "task", DraftBody: "draft", CreatedBy: "agent:mineru"},
		{SkillName: "skill", SourceTask: "", DraftBody: "draft", CreatedBy: "agent:mineru"},
		{SkillName: "skill", SourceTask: "task", DraftBody: "", CreatedBy: "agent:mineru"},
		{SkillName: "skill", SourceTask: "task", DraftBody: "draft", CreatedBy: ""},
	}
	for _, tc := range cases {
		if _, err := svc.SubmitSkillLearningProposal(ctx, tc); err == nil {
			t.Fatalf("expected invalid proposal to fail: %+v", tc)
		}
	}
}

func TestSkillLearningProposal_ListPendingScopedAndNewestFirst(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	base := time.Date(2026, 5, 20, 16, 0, 0, 0, time.UTC)

	_, _ = svc.SubmitSkillLearningProposal(ctx, SkillLearningProposalCreateParams{WorkspaceID: svc.workspaceID, SkillName: "old", SourceTask: "task old", DraftBody: "draft old", CreatedBy: "agent:mineru", CreatedAt: base})
	newer, _ := svc.SubmitSkillLearningProposal(ctx, SkillLearningProposalCreateParams{WorkspaceID: svc.workspaceID, SkillName: "new", SourceTask: "task new", DraftBody: "draft new", CreatedBy: "agent:mineru", CreatedAt: base.Add(time.Minute)})
	_, _ = svc.SubmitSkillLearningProposal(ctx, SkillLearningProposalCreateParams{WorkspaceID: "other-workspace", SkillName: "other", SourceTask: "task other", DraftBody: "draft other", CreatedBy: "agent:mineru", CreatedAt: base.Add(2 * time.Minute)})

	items, err := svc.ListPendingSkillLearningProposals(ctx, SkillLearningProposalQuery{WorkspaceID: svc.workspaceID, Limit: 10})
	if err != nil {
		t.Fatalf("ListPendingSkillLearningProposals: %v", err)
	}
	if len(items.Items) != 2 {
		t.Fatalf("pending count = %d, want 2: %+v", len(items.Items), items.Items)
	}
	if items.Items[0].ProposalID != newer.ProposalID || items.Items[0].SkillName != "new" {
		t.Fatalf("order = %+v, want newest first", items.Items)
	}
}

func TestSkillLearningProposal_ApproveRejectAndDuplicateReview(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	approveRef, _ := svc.SubmitSkillLearningProposal(ctx, SkillLearningProposalCreateParams{SkillName: "approve-me", SourceTask: "task", DraftBody: "draft", CreatedBy: "agent:mineru"})
	approved, err := svc.ApproveSkillLearningProposal(ctx, SkillLearningProposalReviewParams{ProposalID: approveRef.ProposalID, ReviewedBy: "human:juan", ReviewReason: "validated against tests"})
	if err != nil {
		t.Fatalf("ApproveSkillLearningProposal: %v", err)
	}
	if approved.Status != SkillLearningProposalApproved || approved.ReviewedBy != "human:juan" || approved.ReviewReason != "validated against tests" || approved.ReviewedAt == nil {
		t.Fatalf("approved = %+v", approved)
	}
	if _, err := svc.RejectSkillLearningProposal(ctx, SkillLearningProposalReviewParams{ProposalID: approveRef.ProposalID, ReviewedBy: "human:juan", ReviewReason: "second review"}); err == nil {
		t.Fatal("expected duplicate review to fail")
	}

	rejectRef, _ := svc.SubmitSkillLearningProposal(ctx, SkillLearningProposalCreateParams{SkillName: "reject-me", SourceTask: "task", DraftBody: "draft", CreatedBy: "agent:mineru"})
	rejected, err := svc.RejectSkillLearningProposal(ctx, SkillLearningProposalReviewParams{ProposalID: rejectRef.ProposalID, ReviewedBy: "human:juan", ReviewReason: "overfit to one invoice"})
	if err != nil {
		t.Fatalf("RejectSkillLearningProposal: %v", err)
	}
	if rejected.Status != SkillLearningProposalRejected || rejected.ReviewReason != "overfit to one invoice" {
		t.Fatalf("rejected = %+v", rejected)
	}
}

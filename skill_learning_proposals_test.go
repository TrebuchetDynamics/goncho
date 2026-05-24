package goncho

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestSkillLearningProposalPublicFacadeSubmitsListsAndReviewsWithServiceWorkspace(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	createdAt := time.Date(2026, 5, 20, 15, 0, 0, 0, time.UTC)
	ref, err := svc.SubmitSkillLearningProposal(ctx, SkillLearningProposalCreateParams{
		SkillName:  "gormes-git",
		SourceTask: "resolved repeated git blocker with build artifact evidence",
		DraftBody:  "# gormes-git\n\nCheck build artifacts before commit.",
		CreatedBy:  "agent:mineru",
		Evidence: map[string]any{
			"validation": "git diff --check",
			"outcome":    "success",
		},
		CreatedAt: createdAt,
	})
	if err != nil {
		t.Fatalf("Service.SubmitSkillLearningProposal: %v", err)
	}
	if ref.ProposalID == "" || ref.WorkspaceID != svc.workspaceID || ref.Status != SkillLearningProposalPending {
		t.Fatalf("ref = %+v, want pending service-workspace proposal", ref)
	}

	proposal, err := GetSkillLearningProposal(ctx, svc.db, ref.ProposalID)
	if err != nil {
		t.Fatalf("GetSkillLearningProposal facade: %v", err)
	}
	if proposal.SkillName != "gormes-git" || proposal.Status != SkillLearningProposalPending {
		t.Fatalf("proposal = %+v, want pending gormes-git proposal", proposal)
	}
	var evidence map[string]any
	if err := json.Unmarshal(proposal.EvidenceJSON, &evidence); err != nil {
		t.Fatalf("evidence json: %v", err)
	}
	if evidence["validation"] != "git diff --check" || evidence["outcome"] != "success" {
		t.Fatalf("evidence = %#v", evidence)
	}

	listed, err := svc.ListPendingSkillLearningProposals(ctx, SkillLearningProposalQuery{Limit: 10})
	if err != nil {
		t.Fatalf("Service.ListPendingSkillLearningProposals: %v", err)
	}
	if listed.Count != 1 || listed.Items[0].ProposalID != ref.ProposalID {
		t.Fatalf("listed proposals = %+v, want submitted proposal", listed)
	}

	approved, err := svc.ApproveSkillLearningProposal(ctx, SkillLearningProposalReviewParams{ProposalID: ref.ProposalID, ReviewedBy: "human:juan", ReviewReason: "validated against tests"})
	if err != nil {
		t.Fatalf("Service.ApproveSkillLearningProposal: %v", err)
	}
	if approved.Status != SkillLearningProposalApproved || approved.ReviewedBy != "human:juan" || approved.ReviewedAt == nil {
		t.Fatalf("approved proposal = %+v, want approved review metadata", approved)
	}
}

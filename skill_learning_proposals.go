package goncho

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/goncho/internal/skillproposals"
)

type SkillLearningProposalStatus = skillproposals.SkillLearningProposalStatus

const (
	SkillLearningProposalPending  SkillLearningProposalStatus = skillproposals.SkillLearningProposalPending
	SkillLearningProposalApproved SkillLearningProposalStatus = skillproposals.SkillLearningProposalApproved
	SkillLearningProposalRejected SkillLearningProposalStatus = skillproposals.SkillLearningProposalRejected
)

type SkillLearningProposalCreateParams = skillproposals.SkillLearningProposalCreateParams

type SkillLearningProposalReviewParams = skillproposals.SkillLearningProposalReviewParams

type SkillLearningProposalQuery = skillproposals.SkillLearningProposalQuery

type SkillLearningProposalRef = skillproposals.SkillLearningProposalRef

type SkillLearningProposal = skillproposals.SkillLearningProposal

type SkillLearningProposalList = skillproposals.SkillLearningProposalList

func (s *Service) SubmitSkillLearningProposal(ctx context.Context, p SkillLearningProposalCreateParams) (SkillLearningProposalRef, error) {
	if s == nil {
		return SkillLearningProposalRef{}, fmt.Errorf("goncho: nil service")
	}
	if strings.TrimSpace(p.WorkspaceID) == "" {
		p.WorkspaceID = s.workspaceID
	}
	return SubmitSkillLearningProposal(ctx, s.db, p)
}

func SubmitSkillLearningProposal(ctx context.Context, db *sql.DB, p SkillLearningProposalCreateParams) (SkillLearningProposalRef, error) {
	return skillproposals.SubmitSkillLearningProposal(ctx, db, p)
}

func (s *Service) GetSkillLearningProposal(ctx context.Context, proposalID string) (SkillLearningProposal, error) {
	if s == nil {
		return SkillLearningProposal{}, fmt.Errorf("goncho: nil service")
	}
	return GetSkillLearningProposal(ctx, s.db, proposalID)
}

func GetSkillLearningProposal(ctx context.Context, db *sql.DB, proposalID string) (SkillLearningProposal, error) {
	return skillproposals.GetSkillLearningProposal(ctx, db, proposalID)
}

func (s *Service) ListPendingSkillLearningProposals(ctx context.Context, q SkillLearningProposalQuery) (SkillLearningProposalList, error) {
	if s == nil {
		return SkillLearningProposalList{}, fmt.Errorf("goncho: nil service")
	}
	if strings.TrimSpace(q.WorkspaceID) == "" {
		q.WorkspaceID = s.workspaceID
	}
	q.Status = SkillLearningProposalPending
	return ListSkillLearningProposals(ctx, s.db, q)
}

func ListSkillLearningProposals(ctx context.Context, db *sql.DB, q SkillLearningProposalQuery) (SkillLearningProposalList, error) {
	return skillproposals.ListSkillLearningProposals(ctx, db, q)
}

func (s *Service) ApproveSkillLearningProposal(ctx context.Context, p SkillLearningProposalReviewParams) (SkillLearningProposal, error) {
	if s == nil {
		return SkillLearningProposal{}, fmt.Errorf("goncho: nil service")
	}
	return skillproposals.ApproveSkillLearningProposal(ctx, s.db, p)
}

func (s *Service) RejectSkillLearningProposal(ctx context.Context, p SkillLearningProposalReviewParams) (SkillLearningProposal, error) {
	if s == nil {
		return SkillLearningProposal{}, fmt.Errorf("goncho: nil service")
	}
	return skillproposals.RejectSkillLearningProposal(ctx, s.db, p)
}

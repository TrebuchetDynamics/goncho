package goncho

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/goncho/internal/reviewlog"
	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
)

type ReviewKind = reviewlog.ReviewKind

const (
	ReviewKindConflict ReviewKind = reviewlog.ReviewKindConflict
	ReviewKindStale    ReviewKind = reviewlog.ReviewKindStale
)

type ReviewStatus = reviewlog.ReviewStatus

const (
	ReviewStatusOpen     ReviewStatus = reviewlog.ReviewStatusOpen
	ReviewStatusResolved ReviewStatus = reviewlog.ReviewStatusResolved
)

type ReviewResolution = reviewlog.ReviewResolution

const (
	ReviewResolutionAccepted   ReviewResolution = reviewlog.ReviewResolutionAccepted
	ReviewResolutionRejected   ReviewResolution = reviewlog.ReviewResolutionRejected
	ReviewResolutionSuperseded ReviewResolution = reviewlog.ReviewResolutionSuperseded
	ReviewResolutionVerified   ReviewResolution = reviewlog.ReviewResolutionVerified
)

type ReviewItemCreateParams = reviewlog.ReviewItemCreateParams

type ReviewItem = reviewlog.ReviewItem

type ReviewResolutionParams = reviewlog.ReviewResolutionParams

type ReviewQuery = reviewlog.ReviewQuery

type ReviewList = reviewlog.ReviewList

func (s *Service) CreateReviewItem(ctx context.Context, p ReviewItemCreateParams) (ReviewItem, error) {
	if s == nil {
		return ReviewItem{}, fmt.Errorf("goncho: nil service")
	}
	if strings.TrimSpace(p.WorkspaceID) == "" {
		p.WorkspaceID = s.workspaceID
	}
	return CreateReviewItem(ctx, s.db, p)
}

func CreateReviewItem(ctx context.Context, db *sql.DB, p ReviewItemCreateParams) (ReviewItem, error) {
	return reviewlog.CreateReviewItem(ctx, db, p)
}

func (s *Service) ListReviewItems(ctx context.Context, q ReviewQuery) (ReviewList, error) {
	if s == nil {
		return ReviewList{}, fmt.Errorf("goncho: nil service")
	}
	q.WorkspaceID = serviceObservationWorkspace(s.workspaceID, q.WorkspaceID)
	return ListReviewItems(ctx, s.db, q)
}

func ListReviewItems(ctx context.Context, db *sql.DB, q ReviewQuery) (ReviewList, error) {
	return reviewlog.ListReviewItems(ctx, db, q)
}

func (s *Service) ResolveReviewItem(ctx context.Context, p ReviewResolutionParams) (ReviewItem, error) {
	if s == nil {
		return ReviewItem{}, fmt.Errorf("goncho: nil service")
	}
	return ResolveReviewItem(ctx, s.db, p)
}

func ResolveReviewItem(ctx context.Context, db *sql.DB, p ReviewResolutionParams) (ReviewItem, error) {
	return reviewlog.ResolveReviewItem(ctx, db, p)
}

func (s *Service) reviewContextUnavailableEvidence(ctx context.Context, peer string) ([]ContextUnavailableEvidence, error) {
	items, err := s.ListReviewItems(ctx, ReviewQuery{PeerID: peer, Status: ReviewStatusOpen})
	if err != nil {
		return nil, err
	}
	return reviewRequiredUnavailableEvidence(items.Items), nil
}

func reviewItemsForContextSession(items []ReviewItem, sessionKey string) []ReviewItem {
	return reviewlog.ReviewItemsForContextSession(items, sessionKey)
}

func reviewRequiredUnavailableEvidence(items []ReviewItem, sessionKeys ...string) []ContextUnavailableEvidence {
	unavailable := reviewlog.ReviewRequiredUnavailableEvidence(items, sessionKeys...)
	if len(unavailable) == 0 {
		return nil
	}
	return sliceutil.Map(unavailable, func(item reviewlog.ContextUnavailableEvidence) ContextUnavailableEvidence {
		return ContextUnavailableEvidence{
			Field:      item.Field,
			Capability: item.Capability,
			Reason:     item.Reason,
		}
	})
}

package goncho

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestContextReportsReviewWarningMarksOmittedDetails(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	createdAt := time.Date(2026, 5, 19, 15, 0, 0, 0, time.UTC)
	for i, subjectID := range []string{"mem-a", "mem-b", "mem-c", "mem-d"} {
		if _, err := svc.CreateReviewItem(ctx, ReviewItemCreateParams{
			Kind:        ReviewKindStale,
			PeerID:      "peer-omitted",
			SessionKey:  "session-omitted",
			SubjectID:   subjectID,
			Reason:      "memory requires lifecycle review",
			EvidenceIDs: []string{"obs-" + subjectID},
			CreatedAt:   createdAt.Add(time.Duration(i) * time.Second),
		}); err != nil {
			t.Fatalf("CreateReviewItem %s: %v", subjectID, err)
		}
	}

	got, err := svc.Context(ctx, ContextParams{Peer: "peer-omitted", SessionKey: "session-omitted"})
	if err != nil {
		t.Fatalf("Context: %v", err)
	}

	reviewEvidence := reviewRequiredEvidenceFromContext(t, got)
	for _, want := range []string{"4 open review items", "item_details_omitted=1"} {
		if !strings.Contains(reviewEvidence.Reason, want) {
			t.Fatalf("review evidence reason = %q, missing %q", reviewEvidence.Reason, want)
		}
	}
}

func reviewRequiredEvidenceFromContext(t *testing.T, got ContextResult) ContextUnavailableEvidence {
	t.Helper()
	for i := range got.Unavailable {
		if got.Unavailable[i].Capability == "review_required" {
			return got.Unavailable[i]
		}
	}
	t.Fatalf("Unavailable = %#v, want review_required evidence", got.Unavailable)
	return ContextUnavailableEvidence{}
}

func TestContextReportsOpenReviewItemsAsUnavailableEvidence(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	createdAt := time.Date(2026, 5, 19, 14, 0, 0, 0, time.UTC)
	reviewIDs := []string{}
	for _, item := range []ReviewItemCreateParams{
		{
			Kind:        ReviewKindConflict,
			PeerID:      "peer-a",
			SessionKey:  "session-a",
			SubjectID:   "mem-new",
			RelatedID:   "mem-old",
			Reason:      "new memory conflicts with old memory",
			EvidenceIDs: []string{"obs-conflict"},
			CreatedAt:   createdAt,
		},
		{
			Kind:        ReviewKindStale,
			PeerID:      "peer-a",
			SessionKey:  "session-a",
			SubjectID:   "mem-stale",
			Reason:      "memory has not been verified recently",
			EvidenceIDs: []string{"obs-stale"},
			CreatedAt:   createdAt.Add(time.Second),
		},
	} {
		created, err := svc.CreateReviewItem(ctx, item)
		if err != nil {
			t.Fatalf("CreateReviewItem: %v", err)
		}
		reviewIDs = append(reviewIDs, created.ID)
	}
	if _, err := svc.CreateReviewItem(ctx, ReviewItemCreateParams{
		Kind:        ReviewKindConflict,
		WorkspaceID: svc.workspaceID,
		PeerID:      "peer-b",
		SubjectID:   "mem-other-peer",
		Reason:      "other peer conflict",
		EvidenceIDs: []string{"obs-other"},
		CreatedAt:   createdAt.Add(2 * time.Second),
	}); err != nil {
		t.Fatalf("CreateReviewItem other peer: %v", err)
	}
	otherSessionReview, err := svc.CreateReviewItem(ctx, ReviewItemCreateParams{
		Kind:        ReviewKindStale,
		WorkspaceID: svc.workspaceID,
		PeerID:      "peer-a",
		SessionKey:  "session-b",
		SubjectID:   "mem-other-session",
		Reason:      "other session stale memory",
		EvidenceIDs: []string{"obs-other-session"},
		CreatedAt:   createdAt.Add(3 * time.Second),
	})
	if err != nil {
		t.Fatalf("CreateReviewItem other session: %v", err)
	}

	got, err := svc.Context(ctx, ContextParams{Peer: "peer-a", SessionKey: "session-a"})
	if err != nil {
		t.Fatalf("Context: %v", err)
	}

	var reviewEvidence *ContextUnavailableEvidence
	for i := range got.Unavailable {
		if got.Unavailable[i].Capability == "review_required" {
			reviewEvidence = &got.Unavailable[i]
			break
		}
	}
	if reviewEvidence == nil {
		t.Fatalf("Unavailable = %#v, want review_required evidence", got.Unavailable)
	}
	if reviewEvidence.Field != "review" {
		t.Fatalf("review evidence field = %q, want review", reviewEvidence.Field)
	}
	for _, want := range []string{"2 open review items", "conflict=1", "stale=1", "session_key=session-a", "mem-new->mem-old", "mem-stale", "obs-conflict", "obs-stale"} {
		if !strings.Contains(reviewEvidence.Reason, want) {
			t.Fatalf("review evidence reason = %q, missing %q", reviewEvidence.Reason, want)
		}
	}
	for _, want := range reviewIDs {
		if !strings.Contains(reviewEvidence.Reason, want) {
			t.Fatalf("review evidence reason = %q, missing review item id %q", reviewEvidence.Reason, want)
		}
	}
	for _, unwanted := range []string{otherSessionReview.ID, "mem-other-session", "obs-other-session"} {
		if strings.Contains(reviewEvidence.Reason, unwanted) {
			t.Fatalf("review evidence reason = %q, unexpectedly included other-session evidence %q", reviewEvidence.Reason, unwanted)
		}
	}
}

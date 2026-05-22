package goncho

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestContextReportsOpenReviewItemsAsUnavailableEvidence(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	createdAt := time.Date(2026, 5, 19, 14, 0, 0, 0, time.UTC)
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
		if _, err := svc.CreateReviewItem(ctx, item); err != nil {
			t.Fatalf("CreateReviewItem: %v", err)
		}
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
	for _, want := range []string{"2 open review items", "conflict=1", "stale=1", "mem-new->mem-old", "mem-stale"} {
		if !strings.Contains(reviewEvidence.Reason, want) {
			t.Fatalf("review evidence reason = %q, missing %q", reviewEvidence.Reason, want)
		}
	}
}

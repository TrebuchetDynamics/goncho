package goncho

import (
	"context"
	"testing"
	"time"
)

func TestReviewInboxListsOpenConflictAndStaleItems(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	createdAt := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	_, err := svc.CreateReviewItem(ctx, ReviewItemCreateParams{
		Kind:        ReviewKindConflict,
		WorkspaceID: svc.workspaceID,
		PeerID:      "peer-a",
		SessionKey:  "session-a",
		SubjectID:   "mem-new",
		RelatedID:   "mem-old",
		Reason:      "new memory conflicts with old memory",
		EvidenceIDs: []string{"obs-a", "obs-b"},
		CreatedAt:   createdAt,
	})
	if err != nil {
		t.Fatalf("CreateReviewItem conflict: %v", err)
	}
	_, err = svc.CreateReviewItem(ctx, ReviewItemCreateParams{
		Kind:        ReviewKindStale,
		WorkspaceID: svc.workspaceID,
		PeerID:      "peer-a",
		SessionKey:  "session-a",
		SubjectID:   "mem-stale",
		Reason:      "memory has not been verified recently",
		EvidenceIDs: []string{"obs-stale"},
		CreatedAt:   createdAt.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("CreateReviewItem stale: %v", err)
	}
	_, err = svc.CreateReviewItem(ctx, ReviewItemCreateParams{
		Kind:        ReviewKindConflict,
		WorkspaceID: "other-workspace",
		PeerID:      "peer-a",
		SessionKey:  "session-a",
		SubjectID:   "mem-other",
		RelatedID:   "mem-old",
		Reason:      "other workspace conflict",
		EvidenceIDs: []string{"obs-other"},
		CreatedAt:   createdAt.Add(2 * time.Second),
	})
	if err != nil {
		t.Fatalf("CreateReviewItem other workspace: %v", err)
	}

	items, err := svc.ListReviewItems(ctx, ReviewQuery{PeerID: "peer-a", Status: ReviewStatusOpen})
	if err != nil {
		t.Fatalf("ListReviewItems: %v", err)
	}
	if len(items.Items) != 2 {
		t.Fatalf("review item count = %d, want 2: %+v", len(items.Items), items.Items)
	}
	if items.Items[0].Kind != ReviewKindStale || items.Items[1].Kind != ReviewKindConflict {
		t.Fatalf("review item order/kinds = %+v, want newest stale then conflict", items.Items)
	}
	for _, item := range items.Items {
		if item.Status != ReviewStatusOpen {
			t.Fatalf("status = %q, want open", item.Status)
		}
		if item.WorkspaceID != svc.workspaceID || item.PeerID != "peer-a" || item.SessionKey != "session-a" {
			t.Fatalf("scope = %+v, want service workspace peer/session", item)
		}
		if item.Reason == "" || len(item.EvidenceIDs) == 0 {
			t.Fatalf("review item missing reason/evidence: %+v", item)
		}
	}
}

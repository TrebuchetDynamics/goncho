package goncho

import (
	"context"
	"testing"
	"time"
)

func TestReviewPublicFacadeCreatesListsAndResolvesWithServiceWorkspace(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	createdAt := time.Date(2026, 5, 19, 13, 0, 0, 0, time.UTC)
	item, err := svc.CreateReviewItem(ctx, ReviewItemCreateParams{
		Kind:        ReviewKindConflict,
		PeerID:      "peer-a",
		SessionKey:  "session-a",
		SubjectID:   "mem-new",
		RelatedID:   "mem-old",
		Reason:      "new memory conflicts with old memory",
		EvidenceIDs: []string{"obs-a", "obs-b"},
		CreatedAt:   createdAt,
	})
	if err != nil {
		t.Fatalf("Service.CreateReviewItem: %v", err)
	}
	if item.WorkspaceID != svc.workspaceID || item.Status != ReviewStatusOpen {
		t.Fatalf("created review item = %+v, want default workspace and open status", item)
	}

	listed, err := ListReviewItems(ctx, svc.db, ReviewQuery{WorkspaceID: svc.workspaceID, PeerID: "peer-a", Status: ReviewStatusOpen})
	if err != nil {
		t.Fatalf("ListReviewItems facade: %v", err)
	}
	if listed.Count != 1 || listed.Items[0].ID != item.ID {
		t.Fatalf("listed review items = %+v, want created item %s", listed, item.ID)
	}

	resolved, err := ResolveReviewItem(ctx, svc.db, ReviewResolutionParams{
		ID:               item.ID,
		Resolution:       ReviewResolutionVerified,
		ResolvedBy:       "agent:mineru",
		ResolutionReason: "evidence checked through public facade",
		ResolvedAt:       createdAt.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("ResolveReviewItem facade: %v", err)
	}
	if resolved.Status != ReviewStatusResolved || resolved.Resolution != ReviewResolutionVerified || resolved.ResolvedAt == nil {
		t.Fatalf("resolved review item = %+v, want verified resolved item", resolved)
	}
}

package goncho

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestAcceptNegativeEvidenceCandidatesCreatesFormalReviewItems(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	ctx := context.Background()
	failed := false
	for _, id := range []string{"review-fail-1", "review-fail-2"} {
		if _, err := svc.Observe(ctx, ObservationParams{ID: id, Kind: ObservationKindToolError, ProfileID: "mineru", PeerID: "peer-review", SessionKey: "sess-review", Success: &failed, Input: "private failing command", Output: "private failure output", Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(10, 0).UTC()}); err != nil {
			t.Fatalf("Observe %s: %v", id, err)
		}
	}

	created, err := svc.CreateNegativeEvidenceReviewItems(ctx, NegativeEvidenceReviewRequest{PeerID: "peer-review", SessionKey: "sess-review", CreatedAt: time.Unix(20, 0).UTC()})
	if err != nil {
		t.Fatalf("CreateNegativeEvidenceReviewItems: %v", err)
	}
	if len(created) != 1 {
		t.Fatalf("created = %+v, want one review item", created)
	}
	item := created[0]
	if item.Kind != ReviewKindStale || item.Status != ReviewStatusOpen || item.PeerID != "peer-review" || item.SessionKey != "sess-review" {
		t.Fatalf("item = %+v", item)
	}
	if item.SubjectID != "negative-evidence:repeated_tool_failure:bash:sess-review" {
		t.Fatalf("subject_id = %q", item.SubjectID)
	}
	if got := strings.Join(item.EvidenceIDs, ","); got != "review-fail-1,review-fail-2" {
		t.Fatalf("evidence ids = %q", got)
	}
	if !strings.Contains(item.Reason, "negative memory candidate") || !strings.Contains(item.Reason, "verify live state") {
		t.Fatalf("reason = %q", item.Reason)
	}
	if strings.Contains(item.Reason, "private failing command") || strings.Contains(item.Reason, "private failure output") {
		t.Fatalf("review reason leaked raw content: %q", item.Reason)
	}

	again, err := svc.CreateNegativeEvidenceReviewItems(ctx, NegativeEvidenceReviewRequest{PeerID: "peer-review", SessionKey: "sess-review"})
	if err != nil {
		t.Fatalf("CreateNegativeEvidenceReviewItems again: %v", err)
	}
	if len(again) != 0 {
		t.Fatalf("second create = %+v, want idempotent no-op for existing open item", again)
	}
}

package goncho

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNegativeEvidenceReviewsDoNotCollapseDistinctProfiles(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	ctx := context.Background()
	failed := false
	for _, profileID := range []string{"mineru", "yunobo"} {
		for _, id := range []string{profileID + "-fail-1", profileID + "-fail-2"} {
			if _, err := svc.Observe(ctx, ObservationParams{ID: id, Kind: ObservationKindToolError, ProfileID: profileID, PeerID: "peer-review", SessionKey: "sess-review", Success: &failed, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(10, 0).UTC()}); err != nil {
				t.Fatalf("Observe %s: %v", id, err)
			}
		}
	}

	created, err := svc.CreateNegativeEvidenceReviewItems(ctx, NegativeEvidenceReviewRequest{PeerID: "peer-review", SessionKey: "sess-review", CreatedAt: time.Unix(20, 0).UTC()})
	if err != nil {
		t.Fatalf("CreateNegativeEvidenceReviewItems: %v", err)
	}
	if len(created) != 2 {
		t.Fatalf("created = %+v, want separate review items for each profile", created)
	}
	subjects := map[string]bool{}
	profiles := map[string]bool{}
	for _, item := range created {
		if subjects[item.SubjectID] {
			t.Fatalf("duplicate subject id collapsed distinct profile review: %+v", created)
		}
		subjects[item.SubjectID] = true
		profiles[item.SubjectID] = strings.Contains(item.SubjectID, "mineru") || strings.Contains(item.SubjectID, "yunobo")
	}
	if len(subjects) != 2 {
		t.Fatalf("subjects = %+v, want two distinct subject ids", subjects)
	}
	for subject, includesProfile := range profiles {
		if !includesProfile {
			t.Fatalf("subject %q does not expose the profile dimension", subject)
		}
	}
}

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
	if item.SubjectID != "negative-evidence:kind-repeated_tool_failure:workspace-default:profile-mineru:peer-peer-review:session-sess-review:tool-bash" {
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

package reviewlog

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
)

func TestReviewLogListsOpenConflictAndStaleItems(t *testing.T) {
	db := migratedReviewTestDB(t)
	ctx := context.Background()

	createdAt := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	_, err := CreateReviewItem(ctx, db, ReviewItemCreateParams{
		Kind:        ReviewKindConflict,
		WorkspaceID: "workspace-a",
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
	_, err = CreateReviewItem(ctx, db, ReviewItemCreateParams{
		Kind:        ReviewKindStale,
		WorkspaceID: "workspace-a",
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
	_, err = CreateReviewItem(ctx, db, ReviewItemCreateParams{
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

	items, err := ListReviewItems(ctx, db, ReviewQuery{WorkspaceID: "workspace-a", PeerID: "peer-a", Status: ReviewStatusOpen})
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
		if item.WorkspaceID != "workspace-a" || item.PeerID != "peer-a" || item.SessionKey != "session-a" {
			t.Fatalf("scope = %+v, want workspace/peer/session", item)
		}
		if item.Reason == "" || len(item.EvidenceIDs) == 0 {
			t.Fatalf("review item missing reason/evidence: %+v", item)
		}
	}
}

func TestReviewLogAllowsDistinctItemsWithSameCreatedAt(t *testing.T) {
	db := migratedReviewTestDB(t)
	ctx := context.Background()

	createdAt := time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC)
	first, err := CreateReviewItem(ctx, db, ReviewItemCreateParams{
		Kind:        ReviewKindConflict,
		WorkspaceID: "workspace-a",
		PeerID:      "peer-a",
		SubjectID:   "mem-a",
		RelatedID:   "mem-old",
		Reason:      "first memory conflicts with old memory",
		CreatedAt:   createdAt,
	})
	if err != nil {
		t.Fatalf("CreateReviewItem first: %v", err)
	}
	second, err := CreateReviewItem(ctx, db, ReviewItemCreateParams{
		Kind:        ReviewKindStale,
		WorkspaceID: "workspace-a",
		PeerID:      "peer-a",
		SubjectID:   "mem-b",
		Reason:      "second memory is stale",
		CreatedAt:   createdAt,
	})
	if err != nil {
		t.Fatalf("CreateReviewItem second: %v", err)
	}
	if first.ID == second.ID {
		t.Fatalf("review IDs both %q, want distinct IDs for distinct same-timestamp review items", first.ID)
	}

	listed, err := ListReviewItems(ctx, db, ReviewQuery{WorkspaceID: "workspace-a", PeerID: "peer-a", Status: ReviewStatusOpen, Limit: 10})
	if err != nil {
		t.Fatalf("ListReviewItems: %v", err)
	}
	if len(listed.Items) != 2 {
		t.Fatalf("review item count = %d, want 2: %+v", len(listed.Items), listed.Items)
	}
}

func TestReviewLogResolveClosesOpenItemWithReviewerAndReason(t *testing.T) {
	db := migratedReviewTestDB(t)
	ctx := context.Background()

	createdAt := time.Date(2026, 5, 19, 13, 0, 0, 0, time.UTC)
	item, err := CreateReviewItem(ctx, db, ReviewItemCreateParams{
		Kind:        ReviewKindConflict,
		WorkspaceID: "workspace-a",
		PeerID:      "peer-a",
		SessionKey:  "session-a",
		SubjectID:   "mem-new",
		RelatedID:   "mem-old",
		Reason:      "new memory conflicts with old memory",
		EvidenceIDs: []string{"obs-a", "obs-b"},
		CreatedAt:   createdAt,
	})
	if err != nil {
		t.Fatalf("CreateReviewItem: %v", err)
	}

	resolvedAt := createdAt.Add(time.Minute)
	resolved, err := ResolveReviewItem(ctx, db, ReviewResolutionParams{
		ID:               item.ID,
		Resolution:       ReviewResolutionSuperseded,
		ResolvedBy:       "agent:mineru",
		ResolutionReason: "newer memory supersedes old memory after evidence review",
		ResolvedAt:       resolvedAt,
	})
	if err != nil {
		t.Fatalf("ResolveReviewItem: %v", err)
	}
	if resolved.Status != ReviewStatusResolved || resolved.Resolution != ReviewResolutionSuperseded {
		t.Fatalf("resolved state = status %q resolution %q, want resolved/superseded", resolved.Status, resolved.Resolution)
	}
	if resolved.ResolvedBy != "agent:mineru" || resolved.ResolutionReason == "" {
		t.Fatalf("resolved reviewer/reason = %+v", resolved)
	}
	if resolved.ResolvedAt == nil || !resolved.ResolvedAt.Equal(resolvedAt) {
		t.Fatalf("resolved_at = %v, want %s", resolved.ResolvedAt, resolvedAt)
	}

	open, err := ListReviewItems(ctx, db, ReviewQuery{WorkspaceID: "workspace-a", PeerID: "peer-a", Status: ReviewStatusOpen})
	if err != nil {
		t.Fatalf("ListReviewItems open: %v", err)
	}
	if len(open.Items) != 0 {
		t.Fatalf("open review items = %+v, want none", open.Items)
	}

	closed, err := ListReviewItems(ctx, db, ReviewQuery{WorkspaceID: "workspace-a", PeerID: "peer-a", Status: ReviewStatusResolved})
	if err != nil {
		t.Fatalf("ListReviewItems resolved: %v", err)
	}
	if len(closed.Items) != 1 || closed.Items[0].ID != item.ID || closed.Items[0].Resolution != ReviewResolutionSuperseded {
		t.Fatalf("resolved review items = %+v, want superseded item %s", closed.Items, item.ID)
	}
}

func migratedReviewTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", t.TempDir()+"/review.db")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := ensureReviewTable(context.Background(), db); err != nil {
		t.Fatalf("migrate review log: %v", err)
	}
	return db
}

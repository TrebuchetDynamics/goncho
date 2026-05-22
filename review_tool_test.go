package goncho

import (
	"context"
	"testing"
	"time"
)

func TestReviewToolListsAndResolvesReviewItems(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	createdAt := time.Date(2026, 5, 19, 15, 0, 0, 0, time.UTC)
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
		t.Fatalf("CreateReviewItem: %v", err)
	}

	tool := NewReviewTool(svc)
	if tool.Name() != "goncho_review" {
		t.Fatalf("tool name = %q, want goncho_review", tool.Name())
	}
	spec := tool.Spec()
	if spec.Name != "goncho_review" || spec.AuditKind != "review" || spec.Idempotent {
		t.Fatalf("review spec = %+v, want review audit mutating non-idempotent spec", spec)
	}

	listed := executeMemoryTool(t, ctx, tool, `{"action":"list","peer_id":"peer-a","status":"open"}`)
	if stringField(t, listed, "action") != "list" || intField(t, listed, "count") != 1 {
		t.Fatalf("list output = %+v, want one open review item", listed)
	}
	items, ok := listed["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items = %#v, want one item", listed["items"])
	}
	listedItem, ok := items[0].(map[string]any)
	if !ok || listedItem["id"] != item.ID || listedItem["kind"] != string(ReviewKindConflict) {
		t.Fatalf("listed item = %#v, want conflict %s", items[0], item.ID)
	}

	resolved := executeMemoryTool(t, ctx, tool, `{"action":"resolve","id":"`+item.ID+`","resolution":"verified","resolved_by":"agent:mineru","resolution_reason":"evidence checked"}`)
	if stringField(t, resolved, "action") != "resolve" || stringField(t, resolved, "status") != string(ReviewStatusResolved) || stringField(t, resolved, "resolution") != string(ReviewResolutionVerified) {
		t.Fatalf("resolve output = %+v, want resolved verified", resolved)
	}

	open := executeMemoryTool(t, ctx, tool, `{"action":"list","peer_id":"peer-a","status":"open"}`)
	if intField(t, open, "count") != 0 {
		t.Fatalf("open output after resolve = %+v, want no open items", open)
	}
}

func TestReviewToolFiltersReviewChainsBySubjectAndRelatedID(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	createdAt := time.Date(2026, 5, 22, 10, 0, 0, 0, time.UTC)
	wanted, err := svc.CreateReviewItem(ctx, ReviewItemCreateParams{
		Kind:        ReviewKindConflict,
		PeerID:      "peer-a",
		SessionKey:  "session-a",
		SubjectID:   "mem-current",
		RelatedID:   "mem-old",
		Reason:      "newer memory supersedes old memory after evidence review",
		EvidenceIDs: []string{"obs-current", "obs-old"},
		CreatedAt:   createdAt,
	})
	if err != nil {
		t.Fatalf("CreateReviewItem wanted: %v", err)
	}
	for _, item := range []ReviewItemCreateParams{
		{
			Kind:        ReviewKindConflict,
			PeerID:      "peer-a",
			SessionKey:  "session-a",
			SubjectID:   "mem-current",
			RelatedID:   "mem-other",
			Reason:      "same subject but different superseded memory",
			EvidenceIDs: []string{"obs-other"},
			CreatedAt:   createdAt.Add(time.Second),
		},
		{
			Kind:        ReviewKindStale,
			PeerID:      "peer-a",
			SessionKey:  "session-a",
			SubjectID:   "mem-stale",
			RelatedID:   "mem-old",
			Reason:      "same related memory but different subject",
			EvidenceIDs: []string{"obs-stale"},
			CreatedAt:   createdAt.Add(2 * time.Second),
		},
	} {
		if _, err := svc.CreateReviewItem(ctx, item); err != nil {
			t.Fatalf("CreateReviewItem distractor: %v", err)
		}
	}

	tool := NewReviewTool(svc)
	listed := executeMemoryTool(t, ctx, tool, `{"action":"list","peer_id":"peer-a","status":"open","subject_id":"mem-current","related_id":"mem-old"}`)
	if intField(t, listed, "count") != 1 {
		t.Fatalf("filtered list output = %+v, want one matching review-chain item", listed)
	}
	items, ok := listed["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items = %#v, want one item", listed["items"])
	}
	listedItem, ok := items[0].(map[string]any)
	if !ok || listedItem["id"] != wanted.ID || listedItem["subject_id"] != "mem-current" || listedItem["related_id"] != "mem-old" {
		t.Fatalf("listed item = %#v, want review-chain item %s", items[0], wanted.ID)
	}
}

func intField(t *testing.T, m map[string]any, key string) int {
	t.Helper()
	value, ok := m[key]
	if !ok {
		t.Fatalf("missing integer field %q in %+v", key, m)
	}
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		t.Fatalf("field %q = %#v, want integer", key, value)
		return 0
	}
}

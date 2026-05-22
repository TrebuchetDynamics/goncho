package goncho

import (
	"context"
	"encoding/json"
	"strings"
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

func TestReviewToolResolveOutputIncludesResolvedAt(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	item, err := svc.CreateReviewItem(ctx, ReviewItemCreateParams{
		Kind:      ReviewKindStale,
		PeerID:    "peer-audit",
		SubjectID: "mem-stale",
		Reason:    "stale memory requires review audit visibility",
		CreatedAt: time.Date(2026, 5, 22, 13, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("CreateReviewItem: %v", err)
	}

	tool := NewReviewTool(svc)
	resolved := executeMemoryTool(t, ctx, tool, `{"action":"resolve","id":"`+item.ID+`","resolution":"verified","resolved_by":"agent:mineru","resolution_reason":"audit timestamp checked"}`)
	resolvedAtText := stringField(t, resolved, "resolved_at")
	resolvedAt, err := time.Parse(time.RFC3339Nano, resolvedAtText)
	if err != nil {
		t.Fatalf("resolved_at = %q, want RFC3339Nano timestamp: %v", resolvedAtText, err)
	}

	listed, err := svc.ListReviewItems(ctx, ReviewQuery{PeerID: "peer-audit", Status: ReviewStatusResolved})
	if err != nil {
		t.Fatalf("ListReviewItems: %v", err)
	}
	if listed.Count != 1 || listed.Items[0].ID != item.ID || listed.Items[0].ResolvedAt == nil {
		t.Fatalf("resolved review items = %+v, want persisted resolved item %s", listed, item.ID)
	}
	if !listed.Items[0].ResolvedAt.Equal(resolvedAt) {
		t.Fatalf("resolved_at output = %s, persisted resolved_at = %s", resolvedAt.Format(time.RFC3339Nano), listed.Items[0].ResolvedAt.Format(time.RFC3339Nano))
	}
}

func TestReviewToolResolveOutputIncludesReviewChain(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	item, err := svc.CreateReviewItem(ctx, ReviewItemCreateParams{
		Kind:      ReviewKindConflict,
		PeerID:    "peer-chain",
		SubjectID: "mem-current",
		RelatedID: "mem-superseded",
		Reason:    "current memory supersedes an older conflicting claim",
		CreatedAt: time.Date(2026, 5, 22, 14, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("CreateReviewItem: %v", err)
	}

	tool := NewReviewTool(svc)
	resolved := executeMemoryTool(t, ctx, tool, `{"action":"resolve","id":"`+item.ID+`","resolution":"superseded","resolved_by":"agent:mineru","resolution_reason":"chain reviewed"}`)
	if stringField(t, resolved, "subject_id") != "mem-current" || stringField(t, resolved, "related_id") != "mem-superseded" {
		t.Fatalf("resolve output = %+v, want review-chain identifiers", resolved)
	}
}

func TestReviewToolResolveOutputIncludesKind(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	item, err := svc.CreateReviewItem(ctx, ReviewItemCreateParams{
		Kind:      ReviewKindStale,
		PeerID:    "peer-kind",
		SubjectID: "mem-stale-kind",
		Reason:    "stale memory kind should be audit-visible on resolve",
		CreatedAt: time.Date(2026, 5, 22, 15, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("CreateReviewItem: %v", err)
	}

	tool := NewReviewTool(svc)
	resolved := executeMemoryTool(t, ctx, tool, `{"action":"resolve","id":"`+item.ID+`","resolution":"verified","resolved_by":"agent:mineru","resolution_reason":"stale kind reviewed"}`)
	if stringField(t, resolved, "kind") != string(ReviewKindStale) {
		t.Fatalf("resolve output = %+v, want review kind %q", resolved, ReviewKindStale)
	}
}

func TestReviewToolResolveOutputIncludesScope(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	item, err := svc.CreateReviewItem(ctx, ReviewItemCreateParams{
		Kind:       ReviewKindConflict,
		PeerID:     "peer-scope",
		SessionKey: "session-scope",
		SubjectID:  "mem-scoped",
		RelatedID:  "mem-related",
		Reason:     "scoped review item should be audit-visible on resolve",
		CreatedAt:  time.Date(2026, 5, 22, 16, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("CreateReviewItem: %v", err)
	}

	tool := NewReviewTool(svc)
	resolved := executeMemoryTool(t, ctx, tool, `{"action":"resolve","id":"`+item.ID+`","resolution":"verified","resolved_by":"agent:mineru","resolution_reason":"scope reviewed"}`)
	if stringField(t, resolved, "peer_id") != "peer-scope" || stringField(t, resolved, "session_key") != "session-scope" {
		t.Fatalf("resolve output = %+v, want peer/session review scope", resolved)
	}
}

func TestReviewToolResolveOutputIncludesEvidenceIDs(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	item, err := svc.CreateReviewItem(ctx, ReviewItemCreateParams{
		Kind:        ReviewKindConflict,
		PeerID:      "peer-evidence",
		SessionKey:  "session-evidence",
		SubjectID:   "mem-evidence",
		RelatedID:   "mem-conflict",
		Reason:      "evidence identifiers should be audit-visible on resolve",
		EvidenceIDs: []string{"obs-alpha", "obs-beta"},
		CreatedAt:   time.Date(2026, 5, 22, 17, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("CreateReviewItem: %v", err)
	}

	tool := NewReviewTool(svc)
	resolved := executeMemoryTool(t, ctx, tool, `{"action":"resolve","id":"`+item.ID+`","resolution":"verified","resolved_by":"agent:mineru","resolution_reason":"evidence reviewed"}`)
	evidenceIDs, ok := resolved["evidence_ids"].([]any)
	if !ok || len(evidenceIDs) != 2 || evidenceIDs[0] != "obs-alpha" || evidenceIDs[1] != "obs-beta" {
		t.Fatalf("resolve output = %+v, want evidence IDs", resolved)
	}
}

func TestReviewToolResolveOutputIncludesReason(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	const reviewReason = "conflict reason should be audit-visible on resolve"
	item, err := svc.CreateReviewItem(ctx, ReviewItemCreateParams{
		Kind:       ReviewKindConflict,
		PeerID:     "peer-reason",
		SessionKey: "session-reason",
		SubjectID:  "mem-reason",
		RelatedID:  "mem-prior",
		Reason:     reviewReason,
		CreatedAt:  time.Date(2026, 5, 22, 18, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("CreateReviewItem: %v", err)
	}

	tool := NewReviewTool(svc)
	resolved := executeMemoryTool(t, ctx, tool, `{"action":"resolve","id":"`+item.ID+`","resolution":"verified","resolved_by":"agent:mineru","resolution_reason":"reason reviewed"}`)
	if stringField(t, resolved, "reason") != reviewReason {
		t.Fatalf("resolve output = %+v, want review reason %q", resolved, reviewReason)
	}
}

func TestReviewToolResolveOutputIncludesCreatedAt(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	createdAt := time.Date(2026, 5, 22, 19, 0, 0, 123, time.UTC)
	item, err := svc.CreateReviewItem(ctx, ReviewItemCreateParams{
		Kind:       ReviewKindStale,
		PeerID:     "peer-created",
		SessionKey: "session-created",
		SubjectID:  "mem-created",
		Reason:     "created timestamp should be audit-visible on resolve",
		CreatedAt:  createdAt,
	})
	if err != nil {
		t.Fatalf("CreateReviewItem: %v", err)
	}

	tool := NewReviewTool(svc)
	resolved := executeMemoryTool(t, ctx, tool, `{"action":"resolve","id":"`+item.ID+`","resolution":"verified","resolved_by":"agent:mineru","resolution_reason":"created time reviewed"}`)
	createdAtText := stringField(t, resolved, "created_at")
	gotCreatedAt, err := time.Parse(time.RFC3339Nano, createdAtText)
	if err != nil {
		t.Fatalf("created_at = %q, want RFC3339Nano timestamp: %v", createdAtText, err)
	}
	if !gotCreatedAt.Equal(createdAt) {
		t.Fatalf("created_at output = %s, want %s", gotCreatedAt.Format(time.RFC3339Nano), createdAt.Format(time.RFC3339Nano))
	}
}

func TestReviewToolResolveOutputIncludesWorkspaceID(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	item, err := svc.CreateReviewItem(ctx, ReviewItemCreateParams{
		Kind:        ReviewKindConflict,
		WorkspaceID: "workspace-review-audit",
		PeerID:      "peer-workspace",
		SessionKey:  "session-workspace",
		SubjectID:   "mem-workspace",
		RelatedID:   "mem-prior-workspace",
		Reason:      "workspace should be audit-visible on resolve",
		CreatedAt:   time.Date(2026, 5, 22, 20, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("CreateReviewItem: %v", err)
	}

	tool := NewReviewTool(svc)
	resolved := executeMemoryTool(t, ctx, tool, `{"action":"resolve","id":"`+item.ID+`","resolution":"verified","resolved_by":"agent:mineru","resolution_reason":"workspace reviewed"}`)
	if stringField(t, resolved, "workspace_id") != "workspace-review-audit" {
		t.Fatalf("resolve output = %+v, want workspace_id %q", resolved, "workspace-review-audit")
	}
}

func TestReviewToolTreatsBlankStatusAsOpenDefault(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	openItem, err := svc.CreateReviewItem(ctx, ReviewItemCreateParams{
		Kind:      ReviewKindConflict,
		PeerID:    "peer-a",
		SubjectID: "mem-open",
		RelatedID: "mem-old",
		Reason:    "open memory conflict needs review",
		CreatedAt: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("CreateReviewItem open: %v", err)
	}
	resolvedItem, err := svc.CreateReviewItem(ctx, ReviewItemCreateParams{
		Kind:      ReviewKindStale,
		PeerID:    "peer-a",
		SubjectID: "mem-resolved",
		Reason:    "resolved stale memory was already reviewed",
		CreatedAt: time.Date(2026, 5, 22, 11, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("CreateReviewItem resolved: %v", err)
	}
	if _, err := svc.ResolveReviewItem(ctx, ReviewResolutionParams{
		ID:               resolvedItem.ID,
		Resolution:       ReviewResolutionVerified,
		ResolvedBy:       "agent:mineru",
		ResolutionReason: "resolved before default-list check",
	}); err != nil {
		t.Fatalf("ResolveReviewItem: %v", err)
	}

	tool := NewReviewTool(svc)
	listed := executeMemoryTool(t, ctx, tool, `{"action":"list","peer_id":"peer-a","status":"   "}`)
	if intField(t, listed, "count") != 1 {
		t.Fatalf("blank-status list output = %+v, want only open review item", listed)
	}
	items, ok := listed["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items = %#v, want one item", listed["items"])
	}
	listedItem, ok := items[0].(map[string]any)
	if !ok || listedItem["id"] != openItem.ID || listedItem["status"] != string(ReviewStatusOpen) {
		t.Fatalf("listed item = %#v, want open item %s", items[0], openItem.ID)
	}
}

func TestReviewToolRejectsInvalidListFilters(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	if _, err := svc.CreateReviewItem(ctx, ReviewItemCreateParams{
		Kind:      ReviewKindConflict,
		PeerID:    "peer-a",
		SubjectID: "mem-new",
		RelatedID: "mem-old",
		Reason:    "new memory conflicts with old memory",
	}); err != nil {
		t.Fatalf("CreateReviewItem: %v", err)
	}

	tool := NewReviewTool(svc)
	for _, tc := range []struct {
		name    string
		args    string
		wantErr string
	}{
		{
			name:    "status typo",
			args:    `{"action":"list","peer_id":"peer-a","status":"archived"}`,
			wantErr: "status must be open or resolved",
		},
		{
			name:    "kind typo",
			args:    `{"action":"list","peer_id":"peer-a","kind":"supersession"}`,
			wantErr: "kind must be conflict or stale",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tool.Execute(ctx, json.RawMessage(tc.args))
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Execute error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestReviewToolRejectsInvalidResolveResolution(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	item, err := svc.CreateReviewItem(ctx, ReviewItemCreateParams{
		Kind:      ReviewKindConflict,
		PeerID:    "peer-a",
		SubjectID: "mem-new",
		RelatedID: "mem-old",
		Reason:    "new memory conflicts with old memory",
	})
	if err != nil {
		t.Fatalf("CreateReviewItem: %v", err)
	}

	tool := NewReviewTool(svc)
	_, err = tool.Execute(ctx, json.RawMessage(`{"action":"resolve","id":"`+item.ID+`","resolution":"archived","resolved_by":"agent:mineru","resolution_reason":"operator typo"}`))
	if err == nil || !strings.Contains(err.Error(), "resolution must be accepted, rejected, superseded, or verified") {
		t.Fatalf("Execute error = %v, want resolution enum guidance", err)
	}

	open, err := svc.ListReviewItems(ctx, ReviewQuery{PeerID: "peer-a", Status: ReviewStatusOpen})
	if err != nil {
		t.Fatalf("ListReviewItems: %v", err)
	}
	if open.Count != 1 || open.Items[0].ID != item.ID {
		t.Fatalf("open reviews after invalid resolve = %+v, want original item still open", open)
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

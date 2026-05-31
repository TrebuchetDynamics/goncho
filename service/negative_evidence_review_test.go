package goncho

import (
	"context"
	"strconv"
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

func negativeEvidenceReviewEvidenceByTool(items []ReviewItem, tool string) string {
	needle := ":tool-" + tool
	for _, item := range items {
		if strings.Contains(item.SubjectID, needle) {
			return strings.Join(item.EvidenceIDs, ",")
		}
	}
	return ""
}

func TestNegativeEvidenceReviewSubjectIDEscapesAmbiguousDimensions(t *testing.T) {
	left := negativeEvidenceReviewSubjectID(NegativeEvidenceCandidate{
		Kind:        NegativeEvidenceRepeatedToolFailure,
		WorkspaceID: "default",
		ProfileID:   "a:peer-b",
		PeerID:      "c",
		SessionKey:  "sess-review",
		ToolName:    "bash",
	})
	right := negativeEvidenceReviewSubjectID(NegativeEvidenceCandidate{
		Kind:        NegativeEvidenceRepeatedToolFailure,
		WorkspaceID: "default",
		ProfileID:   "a",
		PeerID:      "b:peer-c",
		SessionKey:  "sess-review",
		ToolName:    "bash",
	})
	if left == right {
		t.Fatalf("subject ids collapsed delimiter-bearing dimensions: %q", left)
	}
	if strings.Contains(left, "a:peer-b") || strings.Contains(right, "b:peer-c") {
		t.Fatalf("subject ids expose raw delimiter-bearing dimensions: left=%q right=%q", left, right)
	}

	withSpace := negativeEvidenceReviewSubjectID(NegativeEvidenceCandidate{
		Kind:        NegativeEvidenceRepeatedToolFailure,
		WorkspaceID: "default",
		ProfileID:   "a b",
		PeerID:      "c",
		SessionKey:  "sess-review",
		ToolName:    "bash",
	})
	withHyphen := negativeEvidenceReviewSubjectID(NegativeEvidenceCandidate{
		Kind:        NegativeEvidenceRepeatedToolFailure,
		WorkspaceID: "default",
		ProfileID:   "a-b",
		PeerID:      "c",
		SessionKey:  "sess-review",
		ToolName:    "bash",
	})
	if withSpace == withHyphen {
		t.Fatalf("subject ids collapsed space and hyphen dimensions: %q", withSpace)
	}
	if strings.Contains(withSpace, " ") {
		t.Fatalf("subject id exposes raw spaces: %q", withSpace)
	}
}

func TestNegativeEvidenceReviewsScanBeyondDefaultLimitBeforeCandidateGrouping(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	ctx := context.Background()
	failed := false
	for i := 1; i <= 2; i++ {
		id := "old-bash-deep-fail-" + strconv.Itoa(i)
		if _, err := svc.Observe(ctx, ObservationParams{ID: id, Kind: ObservationKindToolError, ProfileID: "mineru", PeerID: "peer-review-deep-scan", SessionKey: "sess-review-deep-scan", Success: &failed, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(int64(i), 0).UTC()}); err != nil {
			t.Fatalf("Observe %s: %v", id, err)
		}
	}
	for i := 1; i <= 500; i++ {
		id := "new-curl-fail-" + strconv.Itoa(i)
		if _, err := svc.Observe(ctx, ObservationParams{ID: id, Kind: ObservationKindToolError, ProfileID: "mineru", PeerID: "peer-review-deep-scan", SessionKey: "sess-review-deep-scan", Success: &failed, Metadata: map[string]string{"tool_name": "curl"}, ObservedAt: time.Unix(int64(1000+i), 0).UTC()}); err != nil {
			t.Fatalf("Observe %s: %v", id, err)
		}
	}

	created, err := svc.CreateNegativeEvidenceReviewItems(ctx, NegativeEvidenceReviewRequest{PeerID: "peer-review-deep-scan", SessionKey: "sess-review-deep-scan", CreatedAt: time.Unix(2000, 0).UTC()})
	if err != nil {
		t.Fatalf("CreateNegativeEvidenceReviewItems: %v", err)
	}
	if len(created) != 2 {
		t.Fatalf("created = %+v, want both old bash and newer curl repeated-failure candidates before review limiting", created)
	}
	if got := negativeEvidenceReviewEvidenceByTool(created, "bash"); got != "old-bash-deep-fail-1,old-bash-deep-fail-2" {
		t.Fatalf("bash evidence ids = %q, want older repeated failures beyond default scan window", got)
	}
}

func TestNegativeEvidenceReviewsPreferRecentFailuresWhenScanWindowIsFull(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	ctx := context.Background()
	failed := false
	for i := 1; i <= negativeEvidenceObservationScanLimit; i++ {
		id := "old-curl-window-fill-" + strconv.Itoa(i)
		if _, err := svc.Observe(ctx, ObservationParams{ID: id, Kind: ObservationKindToolError, ProfileID: "mineru", PeerID: "peer-review-recent-window", SessionKey: "sess-review-recent-window", Success: &failed, Metadata: map[string]string{"tool_name": "curl"}, ObservedAt: time.Unix(int64(i), 0).UTC()}); err != nil {
			t.Fatalf("Observe %s: %v", id, err)
		}
	}
	for i := 1; i <= 2; i++ {
		id := "recent-bash-window-fail-" + strconv.Itoa(i)
		if _, err := svc.Observe(ctx, ObservationParams{ID: id, Kind: ObservationKindToolError, ProfileID: "mineru", PeerID: "peer-review-recent-window", SessionKey: "sess-review-recent-window", Success: &failed, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(int64(10000+i), 0).UTC()}); err != nil {
			t.Fatalf("Observe %s: %v", id, err)
		}
	}

	created, err := svc.CreateNegativeEvidenceReviewItems(ctx, NegativeEvidenceReviewRequest{PeerID: "peer-review-recent-window", SessionKey: "sess-review-recent-window", CreatedAt: time.Unix(20000, 0).UTC()})
	if err != nil {
		t.Fatalf("CreateNegativeEvidenceReviewItems: %v", err)
	}
	if len(created) < 1 {
		t.Fatalf("created = %+v, want at least one repeated-failure candidate", created)
	}
	if got := negativeEvidenceReviewEvidenceByTool(created, "bash"); got != "recent-bash-window-fail-1,recent-bash-window-fail-2" {
		t.Fatalf("bash evidence ids = %q, want recent repeated failures not dropped by full scan window", got)
	}
}

func TestNegativeEvidenceReviewsFilterObservationScanBeforeDefaultLimit(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	ctx := context.Background()
	failed := false
	for i := 1; i <= 2; i++ {
		id := "old-bash-fail-" + strconv.Itoa(i)
		if _, err := svc.Observe(ctx, ObservationParams{ID: id, Kind: ObservationKindToolError, ProfileID: "mineru", PeerID: "peer-review-scan", SessionKey: "sess-review-scan", Success: &failed, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(int64(i), 0).UTC()}); err != nil {
			t.Fatalf("Observe %s: %v", id, err)
		}
	}
	for i := 1; i <= 500; i++ {
		id := "new-prompt-" + strconv.Itoa(i)
		if _, err := svc.Observe(ctx, ObservationParams{ID: id, Kind: ObservationKindUserPrompt, ProfileID: "mineru", PeerID: "peer-review-scan", SessionKey: "sess-review-scan", Input: "unrelated prompt", ObservedAt: time.Unix(int64(1000+i), 0).UTC()}); err != nil {
			t.Fatalf("Observe %s: %v", id, err)
		}
	}

	created, err := svc.CreateNegativeEvidenceReviewItems(ctx, NegativeEvidenceReviewRequest{PeerID: "peer-review-scan", SessionKey: "sess-review-scan", CreatedAt: time.Unix(2000, 0).UTC()})
	if err != nil {
		t.Fatalf("CreateNegativeEvidenceReviewItems: %v", err)
	}
	if len(created) != 1 {
		t.Fatalf("created = %+v, want older repeated failure not dropped by newer non-failure observations", created)
	}
	if got := strings.Join(created[0].EvidenceIDs, ","); got != "old-bash-fail-1,old-bash-fail-2" {
		t.Fatalf("evidence ids = %q, want older repeated failures", got)
	}
}

func TestNegativeEvidenceReviewsCreatedFromWildcardScanUseCandidateWorkspace(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	ctx := context.Background()
	failed := false
	for _, id := range []string{"wildcard-fail-1", "wildcard-fail-2"} {
		if _, err := svc.Observe(ctx, ObservationParams{ID: id, Kind: ObservationKindToolError, WorkspaceID: "workspace-negative", ProfileID: "mineru", PeerID: "peer-review-wildcard", SessionKey: "sess-review-wildcard", Success: &failed, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(10, 0).UTC()}); err != nil {
			t.Fatalf("Observe %s: %v", id, err)
		}
	}

	created, err := svc.CreateNegativeEvidenceReviewItems(ctx, NegativeEvidenceReviewRequest{WorkspaceID: "*", PeerID: "peer-review-wildcard", SessionKey: "sess-review-wildcard", CreatedAt: time.Unix(20, 0).UTC()})
	if err != nil {
		t.Fatalf("CreateNegativeEvidenceReviewItems: %v", err)
	}
	if len(created) != 1 {
		t.Fatalf("created = %+v, want one review item from wildcard scan", created)
	}
	if created[0].WorkspaceID != "workspace-negative" {
		t.Fatalf("workspace_id = %q, want candidate observation workspace", created[0].WorkspaceID)
	}
	if !strings.Contains(created[0].SubjectID, "workspace-workspace-negative") {
		t.Fatalf("subject_id = %q, want candidate workspace provenance", created[0].SubjectID)
	}
}

func TestNegativeEvidenceReviewsSkipFailuresResolvedByLaterSuccess(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	ctx := context.Background()
	failed := false
	succeeded := true
	for i := 1; i <= 2; i++ {
		id := "resolved-bash-fail-" + strconv.Itoa(i)
		if _, err := svc.Observe(ctx, ObservationParams{ID: id, Kind: ObservationKindToolError, ProfileID: "mineru", PeerID: "peer-review-resolved", SessionKey: "sess-review-resolved", Success: &failed, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(int64(10+i), 0).UTC()}); err != nil {
			t.Fatalf("Observe %s: %v", id, err)
		}
	}
	if _, err := svc.Observe(ctx, ObservationParams{ID: "resolved-bash-success", Kind: ObservationKindToolResult, ProfileID: "mineru", PeerID: "peer-review-resolved", SessionKey: "sess-review-resolved", Success: &succeeded, Metadata: map[string]string{"tool_name": "bash"}, ObservedAt: time.Unix(20, 0).UTC()}); err != nil {
		t.Fatalf("Observe resolved-bash-success: %v", err)
	}

	created, err := svc.CreateNegativeEvidenceReviewItems(ctx, NegativeEvidenceReviewRequest{PeerID: "peer-review-resolved", SessionKey: "sess-review-resolved", CreatedAt: time.Unix(30, 0).UTC()})
	if err != nil {
		t.Fatalf("CreateNegativeEvidenceReviewItems: %v", err)
	}
	if len(created) != 0 {
		t.Fatalf("created = %+v, want no review item for repeated failures resolved by a later scoped success", created)
	}
}

func TestNegativeEvidenceReviewLimitAppliesAfterCandidateGeneration(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	ctx := context.Background()
	failed := false
	for _, tool := range []string{"bash", "curl"} {
		for i := 1; i <= 2; i++ {
			id := tool + "-limit-fail-" + strconv.Itoa(i)
			if _, err := svc.Observe(ctx, ObservationParams{ID: id, Kind: ObservationKindToolError, ProfileID: "mineru", PeerID: "peer-review-limit", SessionKey: "sess-review-limit", Success: &failed, Metadata: map[string]string{"tool_name": tool}, ObservedAt: time.Unix(int64(10+i), 0).UTC()}); err != nil {
				t.Fatalf("Observe %s: %v", id, err)
			}
		}
	}

	created, err := svc.CreateNegativeEvidenceReviewItems(ctx, NegativeEvidenceReviewRequest{PeerID: "peer-review-limit", SessionKey: "sess-review-limit", Limit: 1, CreatedAt: time.Unix(20, 0).UTC()})
	if err != nil {
		t.Fatalf("CreateNegativeEvidenceReviewItems: %v", err)
	}
	if len(created) != 1 {
		t.Fatalf("created = %+v, want one limited candidate after scanning enough observations to prove repetition", created)
	}
	if got := strings.Join(created[0].EvidenceIDs, ","); !strings.Contains(got, "-limit-fail-1") || !strings.Contains(got, "-limit-fail-2") {
		t.Fatalf("evidence ids = %q, want complete repeated-failure evidence for the limited candidate", got)
	}
}

func TestNegativeEvidenceReviewLimitCountsNewItemsAfterExistingOpenItems(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	ctx := context.Background()
	failed := false
	for _, tool := range []string{"curl", "bash"} {
		for i := 1; i <= 2; i++ {
			id := tool + "-existing-skip-fail-" + strconv.Itoa(i)
			observedAt := time.Unix(int64(10+i), 0).UTC()
			if tool == "bash" {
				observedAt = time.Unix(int64(100+i), 0).UTC()
			}
			if _, err := svc.Observe(ctx, ObservationParams{ID: id, Kind: ObservationKindToolError, ProfileID: "mineru", PeerID: "peer-review-existing-skip", SessionKey: "sess-review-existing-skip", Success: &failed, Metadata: map[string]string{"tool_name": tool}, ObservedAt: observedAt}); err != nil {
				t.Fatalf("Observe %s: %v", id, err)
			}
		}
	}

	first, err := svc.CreateNegativeEvidenceReviewItems(ctx, NegativeEvidenceReviewRequest{PeerID: "peer-review-existing-skip", SessionKey: "sess-review-existing-skip", Limit: 1, CreatedAt: time.Unix(200, 0).UTC()})
	if err != nil {
		t.Fatalf("CreateNegativeEvidenceReviewItems first: %v", err)
	}
	if len(first) != 1 || !strings.Contains(first[0].SubjectID, ":tool-bash") {
		t.Fatalf("first created = %+v, want most recent bash candidate", first)
	}

	second, err := svc.CreateNegativeEvidenceReviewItems(ctx, NegativeEvidenceReviewRequest{PeerID: "peer-review-existing-skip", SessionKey: "sess-review-existing-skip", Limit: 1, CreatedAt: time.Unix(300, 0).UTC()})
	if err != nil {
		t.Fatalf("CreateNegativeEvidenceReviewItems second: %v", err)
	}
	if len(second) != 1 || !strings.Contains(second[0].SubjectID, ":tool-curl") {
		t.Fatalf("second created = %+v, want limit to count new review items after skipping existing open bash", second)
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

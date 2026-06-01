package goncho

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRetentionAccessReportClassifiesStaleAndOversizedMemories(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	old := seedRetentionConclusion(t, ctx, svc, "peer-report", "sess-report", "Old retention access memory should become stale.", time.Now().Add(-90*24*time.Hour))
	oversizedContent := "Oversized password token memory " + strings.Repeat("x", 80)
	oversized := seedRetentionConclusion(t, ctx, svc, "peer-report", "sess-report", oversizedContent, time.Now())
	if _, err := svc.CreateReviewItem(ctx, ReviewItemCreateParams{Kind: ReviewKindStale, PeerID: "peer-report", SessionKey: "sess-report", SubjectID: "conclusion:" + itoa64(oversized.ID), Reason: "needs review before use"}); err != nil {
		t.Fatalf("CreateReviewItem: %v", err)
	}

	report, err := svc.RetentionAccessReport(ctx, RetentionAccessReportParams{Now: time.Now(), StaleAfter: 30 * 24 * time.Hour, OversizedBytes: 64, BudgetBytes: 1})
	if err != nil {
		t.Fatalf("RetentionAccessReport: %v", err)
	}
	if report.Mutates || report.Status != "ok" {
		t.Fatalf("report status/mutates = %+v", report)
	}
	if report.Counts["stale"] == 0 || report.Counts["oversized"] == 0 || report.Counts["high_risk"] == 0 || report.Counts["unreviewed"] == 0 || report.Counts["over_budget"] == 0 {
		t.Fatalf("counts = %+v, want stale/oversized/high_risk/unreviewed/over_budget", report.Counts)
	}
	if !retentionReportItemHas(report, "conclusion:"+itoa64(old.ID), "stale") {
		t.Fatalf("items = %+v, want old conclusion classified stale", report.Items)
	}
	if !retentionReportItemHas(report, "conclusion:"+itoa64(oversized.ID), "oversized") || !retentionReportItemHas(report, "conclusion:"+itoa64(oversized.ID), "unreviewed") {
		t.Fatalf("items = %+v, want oversized reviewed conclusion classified", report.Items)
	}
}

func retentionReportItemHas(report RetentionAccessReport, stableID, category string) bool {
	for _, item := range report.Items {
		if item.StableID != stableID {
			continue
		}
		for _, got := range item.Categories {
			if got == category {
				return true
			}
		}
	}
	return false
}

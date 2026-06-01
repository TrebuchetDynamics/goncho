package goncho

import (
	"context"
	"testing"
)

func TestContextSentinelWarnsWhenImportantFactMissing(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	if _, err := svc.UpsertSentinel(ctx, SentinelParams{ID: "sentinel-auth-owner", Peer: "peer-sentinel", Expected: "Mira owns auth escalation"}); err != nil {
		t.Fatalf("UpsertSentinel: %v", err)
	}
	got, err := svc.Context(ctx, ContextParams{Peer: "peer-sentinel", Query: "auth", MaxTokens: 100})
	if err != nil {
		t.Fatalf("Context: %v", err)
	}
	if !contextUnavailableHasCapability(got.Unavailable, "sentinel_missing") {
		t.Fatalf("Unavailable = %+v, want sentinel_missing warning", got.Unavailable)
	}
}

func TestFacetFilterNarrowsViewerMemoryReport(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	keep, err := svc.Conclude(ctx, ConcludeParams{Peer: "peer-facet", Conclusion: "Facet report keeps auth memory."})
	if err != nil {
		t.Fatalf("Conclude keep: %v", err)
	}
	if _, err := svc.Conclude(ctx, ConcludeParams{Peer: "peer-facet", Conclusion: "Facet report omits billing memory."}); err != nil {
		t.Fatalf("Conclude omit: %v", err)
	}
	if err := svc.UpsertMemoryFacet(ctx, FacetParams{StableID: "conclusion:" + itoa64(keep.ID), Facet: "topic", Value: "auth"}); err != nil {
		t.Fatalf("UpsertMemoryFacet: %v", err)
	}
	report, err := svc.ViewerMemoryReport(ctx, "topic", "auth", 10)
	if err != nil {
		t.Fatalf("ViewerMemoryReport: %v", err)
	}
	if len(report.Items) != 1 || report.Items[0].ID != keep.ID || report.Facet != "topic" || !report.ReadOnly {
		t.Fatalf("report = %+v, want one auth-faceted memory", report)
	}
}

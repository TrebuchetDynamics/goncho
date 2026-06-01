package goncho

import (
	"context"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/goncho/memory"
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

func TestViewerMemoryReportRecordsAgentScopeEvidence(t *testing.T) {
	ctx := context.Background()
	store, err := memory.OpenSqlite(t.TempDir()+"/memory.db", 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer func() {
		if err := store.Close(ctx); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}()
	architect := NewService(store.DB(), Config{WorkspaceID: "default", AgentRoleID: "architect", AgentScopeMode: AgentScopeIsolated}, nil)
	reviewer := NewService(store.DB(), Config{WorkspaceID: "default", AgentRoleID: "reviewer", AgentScopeMode: AgentScopeIsolated}, nil)
	if _, err := architect.Conclude(ctx, ConcludeParams{Peer: "peer-memory", Conclusion: "Architect memory visible."}); err != nil {
		t.Fatalf("architect conclude: %v", err)
	}
	if _, err := reviewer.Conclude(ctx, ConcludeParams{Peer: "peer-memory", Conclusion: "Reviewer memory hidden from architect viewer."}); err != nil {
		t.Fatalf("reviewer conclude: %v", err)
	}
	report, err := architect.ViewerMemoryReport(ctx, "", "", 10)
	if err != nil {
		t.Fatalf("ViewerMemoryReport: %v", err)
	}
	if report.AgentScope == nil || report.AgentScope.Mode != AgentScopeIsolated || report.AgentScope.AgentID != "architect" || !report.AgentScope.Applied {
		t.Fatalf("agent scope evidence = %+v", report.AgentScope)
	}
	if len(report.Items) != 1 || !strings.Contains(report.Items[0].Content, "Architect") {
		t.Fatalf("viewer items = %+v", report.Items)
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

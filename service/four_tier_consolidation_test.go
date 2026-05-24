package goncho

import (
	"context"
	"slices"
	"strings"
	"testing"
)

func TestExecuteFourTierConsolidationWritesTieredMemoriesWithProvenance(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}

	ctx := context.Background()
	if _, err := svc.CreateMessages(ctx, CreateMessagesParams{SessionKey: "sess-tier", Messages: []CreateMessage{
		{Peer: "peer-tier", Role: "user", Content: "We investigated the auth outage. Decision: use SQLite cache for local recall."},
		{Peer: "peer-tier", Role: "assistant", Content: "Procedure: before deploy, run go test ./... and check hook capture audit."},
	}}); err != nil {
		t.Fatalf("CreateMessages: %v", err)
	}
	if _, err := svc.Conclude(ctx, ConcludeParams{Peer: "peer-tier", SessionKey: "sess-tier", Conclusion: "Mira owns the auth service."}); err != nil {
		t.Fatal(err)
	}

	result, err := svc.ExecuteFourTierConsolidation(ctx, FourTierConsolidationParams{Peer: "peer-tier", SessionKey: "sess-tier"})
	if err != nil {
		t.Fatalf("ExecuteFourTierConsolidation: %v", err)
	}
	if got, want := consolidationTiers(result.Items), []MemoryConsolidationTier{ConsolidationTierWorking, ConsolidationTierEpisodic, ConsolidationTierSemantic, ConsolidationTierProcedural}; !slices.Equal(got, want) {
		t.Fatalf("tiers = %v, want %v", got, want)
	}
	for _, item := range result.Items {
		if item.MemoryID == 0 || item.Content == "" {
			t.Fatalf("consolidated item missing persisted memory/content: %+v", item)
		}
		if len(item.Provenance) == 0 {
			t.Fatalf("consolidated item %s missing provenance", item.Tier)
		}
		if item.Provenance[0].Kind != "consolidation" || item.Provenance[0].Metadata["session_key"] != "sess-tier" {
			t.Fatalf("provenance = %+v, want consolidation session provenance", item.Provenance)
		}
	}
	if result.Items[3].Tier != ConsolidationTierProcedural || !containsSubstring(result.Items[3].Content, "before deploy") {
		t.Fatalf("procedural item = %+v, want extracted procedure", result.Items[3])
	}

	search, err := svc.Search(ctx, SearchParams{Peer: "peer-tier", Query: "before deploy", SessionKey: "sess-tier", Limit: 5})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if !searchContains(search.Results, "procedural") {
		t.Fatalf("search results = %+v, want persisted procedural consolidation", search.Results)
	}
}

func consolidationTiers(items []ConsolidatedMemory) []MemoryConsolidationTier {
	out := make([]MemoryConsolidationTier, 0, len(items))
	for _, item := range items {
		out = append(out, item.Tier)
	}
	return out
}

func containsSubstring(value, needle string) bool {
	return strings.Contains(value, needle)
}

func searchContains(results []SearchHit, needle string) bool {
	for _, result := range results {
		if strings.Contains(result.Content, needle) {
			return true
		}
	}
	return false
}

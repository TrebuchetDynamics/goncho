package goncho

import "testing"

func TestMergeVectorRecallCandidatesDeduplicatesGeneratedIDsForTrimEquivalentHits(t *testing.T) {
	hits := []VectorSearchHit{
		{SourceType: "memory", SessionID: "session-1", Content: "semantic owner fact", Score: 0.40, Metadata: map[string]string{"rank": "low"}},
		{SourceType: " memory ", SessionID: " session-1 ", Content: "  semantic owner fact  ", Score: 0.90, Metadata: map[string]string{"rank": "high"}},
	}

	items := mergeVectorRecallCandidates(nil, hits, "agent-a", "team", nil)

	if len(items) != 1 {
		t.Fatalf("candidate count = %d, want trim-equivalent generated vector IDs merged: %+v", len(items), items)
	}
	if got := items[0].MemoryID; got == "" || got != semanticMemoryID(hits[0]) || got != semanticMemoryID(hits[1]) {
		t.Fatalf("merged memory ID = %q, want shared generated semantic ID for trim-equivalent hits", got)
	}
	if len(items[0].Provenance) != 1 {
		t.Fatalf("merged provenance count = %d, want one semantic evidence item", len(items[0].Provenance))
	}
	if got := items[0].Provenance[0].Score; got != 0.90 {
		t.Fatalf("merged semantic score = %v, want strongest vector score", got)
	}
	if got := items[0].Provenance[0].Metadata["rank"]; got != "high" {
		t.Fatalf("merged metadata rank = %q, want strongest hit metadata to win", got)
	}
}

func TestMergeVectorRecallCandidatesUsesStableTrimmedMemoryIDForBaseMatches(t *testing.T) {
	base := []RecallCandidate{{
		MemoryID:   " mem-owner ",
		Content:    "lexical owner fact",
		Provenance: []EvidenceItem{{Kind: "keyword", Source: "fts", ID: "mem-owner", Score: 0.70}},
	}}
	hits := []VectorSearchHit{{MemoryID: "mem-owner", SourceType: "memory", Content: "semantic owner fact", Score: 0.95}}

	items := mergeVectorRecallCandidates(base, hits, "agent-a", "team", nil)

	if len(items) != 1 {
		t.Fatalf("candidate count = %d, want vector hit merged into trim-equivalent base memory ID: %+v", len(items), items)
	}
	if len(items[0].Provenance) != 2 {
		t.Fatalf("merged provenance = %+v, want lexical and semantic evidence retained", items[0].Provenance)
	}
}

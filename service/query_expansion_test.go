package goncho

import (
	"context"
	"testing"
)

func TestSearchAndRecallUseQueryExpansionWithProvenance(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	if _, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "peer-expansion",
		Conclusion: "Mira owns the authentication service and rotates login credentials.",
		SessionKey: "sess-expansion",
	}); err != nil {
		t.Fatal(err)
	}

	search, err := svc.Search(ctx, SearchParams{
		Peer:       "peer-expansion",
		Query:      "signin",
		SessionKey: "sess-expansion",
		Limit:      3,
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(search.Results) == 0 {
		t.Fatal("search returned no results; want synonym-expanded signin -> login/authentication hit")
	}
	if !searchHitHasEvidence(search.Results[0], "query_expansion") {
		t.Fatalf("search provenance = %+v, want query_expansion evidence", search.Results[0].Provenance)
	}

	trace, err := svc.Recall(ctx, RecallQuery{
		Peer:       "peer-expansion",
		Query:      "signin",
		SessionKey: "sess-expansion",
		Limit:      3,
	})
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}
	if len(trace.Selected) == 0 {
		t.Fatal("recall selected no results; want synonym-expanded signin -> login/authentication hit")
	}
	selected := trace.Selected[0].Candidate
	if !recallCandidateHasEvidence(selected, "query_expansion", "signin") {
		t.Fatalf("recall provenance = %+v, want query_expansion evidence", selected.Provenance)
	}
	if !recallCandidateHasEvidence(selected, "keyword", "expanded:signin") {
		t.Fatalf("recall provenance = %+v, want expanded keyword evidence", selected.Provenance)
	}
}

func searchHitHasEvidence(hit SearchHit, kind string) bool {
	for _, evidence := range hit.Provenance {
		if evidence.Kind == kind {
			return true
		}
	}
	return false
}

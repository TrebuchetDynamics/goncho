package goncho

import (
	"context"
	"errors"
	"strconv"
	"testing"
)

func TestSearchUsesOptionalVectorStoreForSemanticLane(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	svc.vectorStore = &fakeVectorStore{hits: []VectorSearchHit{{MemoryID: "semantic-orchid", SourceType: "conclusion", Content: "Mira stores the rare orchid archive in the blue vault.", Score: 0.91}}}
	svc.providerRegistry = NewProviderHealthRegistry(ProviderResilienceConfig{}, svc.vectorStore)

	got, err := svc.Search(context.Background(), SearchParams{Peer: "peer-semantic-search", Query: "botanical dossier location", Limit: 3})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got.Results) != 1 || got.Results[0].Content != "Mira stores the rare orchid archive in the blue vault." {
		t.Fatalf("semantic Search results = %+v", got.Results)
	}
	if !searchHitHasEvidence(got.Results[0], "semantic") {
		t.Fatalf("semantic Search provenance = %+v, want semantic evidence", got.Results[0].Provenance)
	}
}

func TestSearchVectorCandidatesDoNotOverflowLimit(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	if _, err := svc.Conclude(ctx, ConcludeParams{Peer: "peer-fusion", Conclusion: "orchid lexical candidate one", Scope: "benchmark"}); err != nil {
		t.Fatalf("conclude lexical one: %v", err)
	}
	if _, err := svc.Conclude(ctx, ConcludeParams{Peer: "peer-fusion", Conclusion: "orchid lexical candidate two", Scope: "benchmark"}); err != nil {
		t.Fatalf("conclude lexical two: %v", err)
	}
	svc.vectorStore = &fakeVectorStore{hits: []VectorSearchHit{{MemoryID: "semantic-dossier", SourceType: "conclusion", Content: "botanical dossier lives in the blue notebook", Score: 0.99}}}
	svc.providerRegistry = NewProviderHealthRegistry(ProviderResilienceConfig{}, svc.vectorStore)

	got, err := svc.Search(ctx, SearchParams{Peer: "peer-fusion", Query: "orchid", Limit: 2})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got.Results) != 2 {
		t.Fatalf("hybrid Search results len = %d, want limit 2: %+v", len(got.Results), got.Results)
	}
}

func TestSearchRerankerIsOptInByDefault(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	first, err := svc.Conclude(ctx, ConcludeParams{Peer: "peer-rerank-default", Conclusion: "orchid candidate first", Scope: "benchmark"})
	if err != nil {
		t.Fatalf("conclude first: %v", err)
	}
	if _, err := svc.Conclude(ctx, ConcludeParams{Peer: "peer-rerank-default", Conclusion: "orchid candidate second", Scope: "benchmark"}); err != nil {
		t.Fatalf("conclude second: %v", err)
	}

	got, err := svc.Search(ctx, SearchParams{Peer: "peer-rerank-default", Query: "orchid", Limit: 2})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got.Results) != 2 || got.Results[0].ID != first.ID {
		t.Fatalf("default Search results = %+v, want original ranking without opt-in reranker", got.Results)
	}
}

func TestSearchUsesOptionalRerankerWhenConfigured(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	first, err := svc.Conclude(ctx, ConcludeParams{Peer: "peer-rerank", Conclusion: "orchid candidate first", Scope: "benchmark"})
	if err != nil {
		t.Fatalf("conclude first: %v", err)
	}
	second, err := svc.Conclude(ctx, ConcludeParams{Peer: "peer-rerank", Conclusion: "orchid candidate second", Scope: "benchmark"})
	if err != nil {
		t.Fatalf("conclude second: %v", err)
	}
	svc.searchReranker = fakeSearchReranker{scores: map[string]float64{strconv.FormatInt(first.ID, 10): 0.1, strconv.FormatInt(second.ID, 10): 0.9}}

	got, err := svc.Search(ctx, SearchParams{Peer: "peer-rerank", Query: "orchid", Limit: 2})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got.Results) != 2 || got.Results[0].ID != second.ID {
		t.Fatalf("reranked Search results = %+v, want second candidate first", got.Results)
	}
}

func TestSearchFallsBackWhenOptionalRerankerFails(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	first, err := svc.Conclude(ctx, ConcludeParams{Peer: "peer-rerank-fallback", Conclusion: "orchid candidate first", Scope: "benchmark"})
	if err != nil {
		t.Fatalf("conclude first: %v", err)
	}
	if _, err := svc.Conclude(ctx, ConcludeParams{Peer: "peer-rerank-fallback", Conclusion: "orchid candidate second", Scope: "benchmark"}); err != nil {
		t.Fatalf("conclude second: %v", err)
	}
	svc.searchReranker = fakeSearchReranker{err: errors.New("reranker unavailable")}

	got, err := svc.Search(ctx, SearchParams{Peer: "peer-rerank-fallback", Query: "orchid", Limit: 2})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got.Results) != 2 || got.Results[0].ID != first.ID {
		t.Fatalf("fallback Search results = %+v, want original ranking", got.Results)
	}
}

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

type fakeSearchReranker struct {
	scores map[string]float64
	err    error
}

func (f fakeSearchReranker) RerankSearch(_ context.Context, _ string, candidates []SearchRerankCandidate) ([]SearchRerankScore, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make([]SearchRerankScore, 0, len(candidates))
	for _, candidate := range candidates {
		out = append(out, SearchRerankScore{ID: candidate.ID, Score: f.scores[candidate.ID]})
	}
	return out, nil
}

func searchHitHasEvidence(hit SearchHit, kind string) bool {
	for _, evidence := range hit.Provenance {
		if evidence.Kind == kind {
			return true
		}
	}
	return false
}

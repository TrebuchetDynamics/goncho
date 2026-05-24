package goncho

import (
	"context"
	"slices"
	"testing"

	"github.com/TrebuchetDynamics/goncho/memory"
)

func TestServiceRecallReturnsScoredTraceWithProvenance(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	_, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "peer-recall",
		Conclusion: "The user prefers deterministic scoring over LLM judges.",
		SessionKey: "sess-recall",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Conclude(ctx, ConcludeParams{
		Peer:       "peer-recall",
		Conclusion: "Graph expansion improves multi-hop recall.",
		SessionKey: "sess-recall",
	})
	if err != nil {
		t.Fatal(err)
	}

	trace, err := svc.Recall(ctx, RecallQuery{
		Peer:       "peer-recall",
		Query:      "deterministic scoring",
		SessionKey: "sess-recall",
		Limit:      5,
	})
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}
	if trace.PipelineVersion == "" {
		t.Fatal("trace missing pipeline_version")
	}
	if trace.Query.Peer != "peer-recall" {
		t.Fatalf("trace query peer = %q, want peer-recall", trace.Query.Peer)
	}
	if len(trace.Candidates) == 0 {
		t.Fatal("trace has no scored candidates")
	}
	if len(trace.Selected) == 0 {
		t.Fatal("trace has no selected candidates")
	}
	for _, item := range trace.Selected {
		if item.Candidate.MemoryID == "" {
			t.Fatal("selected candidate missing memory_id")
		}
		if len(item.Candidate.Provenance) == 0 {
			t.Fatalf("selected candidate %s missing provenance", item.Candidate.MemoryID)
		}
		if item.Score.FinalScore <= 0 {
			t.Fatalf("selected candidate %s final_score = %v, want > 0", item.Candidate.MemoryID, item.Score.FinalScore)
		}
	}
}

func TestServiceRecallEmptyQueryReturnsNoCandidates(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	trace, err := svc.Recall(context.Background(), RecallQuery{
		Peer:  "peer-recall-empty",
		Query: "",
		Limit: 5,
	})
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}
	if len(trace.Selected) != 0 {
		t.Fatalf("empty query selected = %d, want 0", len(trace.Selected))
	}
}

func TestServiceRecallPeerIsRequired(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	_, err := svc.Recall(context.Background(), RecallQuery{
		Query: "something",
		Limit: 5,
	})
	if err == nil {
		t.Fatal("Recall with empty peer should return an error")
	}
}

func TestServiceRecallDefaultsWorkspaceFromService(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	_, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "peer-recall-ws",
		Conclusion: "Service default workspace is used when query omits it.",
		SessionKey: "sess-recall-ws",
	})
	if err != nil {
		t.Fatal(err)
	}

	trace, err := svc.Recall(ctx, RecallQuery{
		Peer:       "peer-recall-ws",
		Query:      "workspace default",
		SessionKey: "sess-recall-ws",
		Limit:      3,
	})
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}
	if trace.Query.WorkspaceID != "default" {
		t.Fatalf("trace workspace = %q, want default", trace.Query.WorkspaceID)
	}
	if len(trace.Selected) == 0 {
		t.Fatal("expected selected candidates from service default workspace")
	}
}

func TestServiceRecallTraceIncludesReplayContract(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	_, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "peer-recall-replay",
		Conclusion: "Recall trace supports deterministic replay.",
		SessionKey: "sess-recall-replay",
	})
	if err != nil {
		t.Fatal(err)
	}

	trace, err := svc.Recall(ctx, RecallQuery{
		Peer:       "peer-recall-replay",
		Query:      "deterministic replay",
		SessionKey: "sess-recall-replay",
		Limit:      3,
	})
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}
	replay := BuildRecallReplay(trace)
	if replay.Service != "goncho" {
		t.Fatalf("replay service = %q, want goncho", replay.Service)
	}
	if replay.TraceID == "" {
		t.Fatal("replay missing trace_id")
	}
	if len(replay.Events) == 0 {
		t.Fatal("replay has no events")
	}
	if replay.ReplayContract != "deterministic_replay_from_recall_trace" {
		t.Fatalf("replay contract = %q, want deterministic_replay_from_recall_trace", replay.ReplayContract)
	}
}

func TestServiceRecallUsesOptionalVectorStoreForSemanticRRF(t *testing.T) {
	ctx := context.Background()
	store, err := memory.OpenSqlite(t.TempDir()+"/memory.db", 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer func() {
		if err := store.Close(context.Background()); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}()

	vectorStore := &fakeVectorStore{
		hits: []VectorSearchHit{{
			MemoryID:   "vec-flower-archive",
			SourceType: "conclusion",
			Content:    "Maya hid the flower archive reference in the blue vault.",
			SessionID:  "sess-vector",
			AgentID:    "gormes",
			ScopeID:    MemoryScopeWorkspace,
			Score:      0.97,
		}},
	}
	svc := NewService(store.DB(), Config{
		WorkspaceID:    "default",
		ObserverPeerID: "gormes",
		RecentMessages: 4,
		VectorStore:    vectorStore,
	}, nil)
	if _, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "peer-vector",
		Conclusion: "A lexical distractor mentions rare orchid marker but not the archive location.",
		SessionKey: "sess-vector",
	}); err != nil {
		t.Fatal(err)
	}

	trace, err := svc.Recall(ctx, RecallQuery{
		Peer:       "peer-vector",
		Query:      "rare orchid marker location",
		SessionKey: "sess-vector",
		Limit:      2,
	})
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}
	if !slices.Contains(selectedRecallIDs(trace), "vec-flower-archive") {
		t.Fatalf("selected IDs = %v, want vector semantic hit included", selectedRecallIDs(trace))
	}
	selected, ok := selectedRecallByID(trace, "vec-flower-archive")
	if !ok {
		t.Fatal("missing selected vector candidate")
	}
	if selected.Score.SemanticScore <= 0 || selected.Score.RRFScore <= 0 {
		t.Fatalf("vector selected score = %+v, want semantic score and RRF contribution", selected.Score)
	}
	if !recallCandidateHasEvidence(selected.Candidate, "semantic", "vec-flower-archive") {
		t.Fatalf("vector candidate provenance = %+v, want semantic evidence", selected.Candidate.Provenance)
	}
	if len(vectorStore.queries) != 1 {
		t.Fatalf("vector queries = %d, want one optional vector search", len(vectorStore.queries))
	}
	if vectorStore.queries[0].Query != "rare orchid marker location" || vectorStore.queries[0].Peer != "peer-vector" {
		t.Fatalf("vector query = %+v, want recall query propagated", vectorStore.queries[0])
	}
}

func TestServiceRecallProjectorRoundTripsToSearchResult(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	_, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "peer-recall-project",
		Conclusion: "Recall projector converts trace to flat search results.",
		SessionKey: "sess-recall-project",
	})
	if err != nil {
		t.Fatal(err)
	}

	trace, err := svc.Recall(ctx, RecallQuery{
		Peer:       "peer-recall-project",
		Query:      "projector flat search",
		SessionKey: "sess-recall-project",
		Limit:      3,
	})
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}
	projector := RecallProjector{}
	searchResult := projector.ProjectSearch(trace)
	if searchResult.Peer != "peer-recall-project" {
		t.Fatalf("projected peer = %q, want peer-recall-project", searchResult.Peer)
	}
	if len(searchResult.Results) == 0 {
		t.Fatal("projected search has no results")
	}
}

type fakeVectorStore struct {
	hits    []VectorSearchHit
	queries []VectorSearchQuery
}

func (f *fakeVectorStore) Search(ctx context.Context, query VectorSearchQuery) ([]VectorSearchHit, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.queries = append(f.queries, query)
	out := make([]VectorSearchHit, len(f.hits))
	copy(out, f.hits)
	return out, nil
}

func selectedRecallByID(trace RecallTrace, memoryID string) (ScoredRecallCandidate, bool) {
	for _, item := range trace.Selected {
		if item.Candidate.MemoryID == memoryID {
			return item, true
		}
	}
	return ScoredRecallCandidate{}, false
}

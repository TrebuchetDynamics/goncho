package goncho

import (
	"context"
	"slices"
	"strconv"
	"testing"
	"time"

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

func TestServiceRecallUsesConclusionUpdatedAtForRecencyVoice(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	oldResult, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "peer-recall-recency",
		Conclusion: "Recency marker belongs to the older conclusion.",
		SessionKey: "sess-recall-recency",
	})
	if err != nil {
		t.Fatalf("conclude old: %v", err)
	}
	newResult, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "peer-recall-recency",
		Conclusion: "Recency marker belongs to the newer conclusion.",
		SessionKey: "sess-recall-recency",
	})
	if err != nil {
		t.Fatalf("conclude new: %v", err)
	}
	oldUpdatedAt := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	newUpdatedAt := oldUpdatedAt.Add(24 * time.Hour)
	if _, err := svc.db.ExecContext(ctx, `UPDATE goncho_conclusions SET updated_at = ? WHERE id = ?`, oldUpdatedAt.Unix(), oldResult.ID); err != nil {
		t.Fatalf("set old updated_at: %v", err)
	}
	if _, err := svc.db.ExecContext(ctx, `UPDATE goncho_conclusions SET updated_at = ? WHERE id = ?`, newUpdatedAt.Unix(), newResult.ID); err != nil {
		t.Fatalf("set new updated_at: %v", err)
	}

	trace, err := svc.RecallWithScoringConfig(ctx, RecallQuery{
		Peer:       "peer-recall-recency",
		Query:      "recency marker",
		SessionKey: "sess-recall-recency",
		Limit:      1,
	}, RecallScoringConfig{
		Version: "recency-regression-v1",
		Weights: map[string]float64{"recency": 1},
		RRFK:    60,
	})
	if err != nil {
		t.Fatalf("RecallWithScoringConfig: %v", err)
	}
	newMemoryID := strconv.FormatInt(newResult.ID, 10)
	if !slices.Equal(selectedRecallIDs(trace), []string{newMemoryID}) {
		t.Fatalf("selected IDs = %v, want newer conclusion %s selected by recency", selectedRecallIDs(trace), newMemoryID)
	}
	if len(trace.Selected) != 1 || !trace.Selected[0].Candidate.CreatedAt.Equal(newUpdatedAt) || trace.Selected[0].Score.RecencyScore <= 0 {
		t.Fatalf("selected = %+v, want updated_at propagated as non-zero recency evidence", trace.Selected)
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

func TestServiceRecallSourceVectorRunsSemanticLaneWithoutConclusions(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	vectorStore := &fakeVectorStore{hits: []VectorSearchHit{{
		MemoryID:   "vec-only",
		SourceType: "vector",
		Content:    "semantic-only recall content",
		SessionID:  "sess-vector-only",
		Score:      0.99,
	}}}
	svc.vectorStore = vectorStore
	svc.providerRegistry = NewProviderHealthRegistry(ProviderResilienceConfig{}, svc.vectorStore)

	trace, err := svc.Recall(context.Background(), RecallQuery{Peer: "peer-vector-only", Query: "semantic only", Sources: []string{"vector"}, Limit: 1})
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}
	if len(vectorStore.queries) != 1 {
		t.Fatalf("vector queries = %d, want source=vector to run semantic lane", len(vectorStore.queries))
	}
	if got := selectedRecallIDs(trace); !slices.Equal(got, []string{"vec-only"}) {
		t.Fatalf("selected IDs = %v, want vector-only candidate", got)
	}
}

func TestServiceRecallSourceConclusionRejectsUntypedVectorHit(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	vectorStore := &fakeVectorStore{hits: []VectorSearchHit{{
		MemoryID: "recall-untyped-vector",
		Content:  "untyped vector recall content should not satisfy conclusion source",
		Score:    0.99,
	}}}
	svc.vectorStore = vectorStore
	svc.providerRegistry = NewProviderHealthRegistry(ProviderResilienceConfig{}, svc.vectorStore)

	trace, err := svc.Recall(context.Background(), RecallQuery{Peer: "peer-recall-untyped-vector-source", Query: "untyped source", Sources: []string{"conclusion"}, Limit: 1})
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}
	if len(vectorStore.queries) != 1 {
		t.Fatalf("vector queries = %d, want conclusion source to query conclusion vector lane", len(vectorStore.queries))
	}
	if got := selectedRecallIDs(trace); len(got) != 0 {
		t.Fatalf("selected IDs = %v, want untyped vector hit treated as source=vector, not conclusion", got)
	}
}

func TestServiceRecallDeduplicatesVectorHitsByMemoryID(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	svc.vectorStore = &fakeVectorStore{hits: []VectorSearchHit{
		{MemoryID: "vec-orchid", SourceType: "conclusion", Content: "lower scoring stale orchid recall content", SessionID: "sess-vector-dedupe", Score: 0.41},
		{MemoryID: "vec-orchid", SourceType: "conclusion", Content: "higher scoring fresh orchid recall content", SessionID: "sess-vector-dedupe", Score: 0.95},
	}}
	svc.providerRegistry = NewProviderHealthRegistry(ProviderResilienceConfig{}, svc.vectorStore)

	trace, err := svc.Recall(context.Background(), RecallQuery{Peer: "peer-vector-dedupe", Query: "orchid", Limit: 5})
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}
	selected, ok := selectedRecallByID(trace, "vec-orchid")
	if !ok {
		t.Fatalf("selected IDs = %v, want vec-orchid", selectedRecallIDs(trace))
	}
	if selected.Candidate.Content != "higher scoring fresh orchid recall content" {
		t.Fatalf("deduped vector recall content = %q, want highest-scoring hit retained", selected.Candidate.Content)
	}
	if !recallCandidateHasEvidence(selected.Candidate, "semantic", "vec-orchid") {
		t.Fatalf("deduped vector recall provenance = %+v, want semantic evidence", selected.Candidate.Provenance)
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

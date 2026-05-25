package goncho

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/goncho/memory"
)

func TestLocalVectorIndexPersistsDiagnosticsAndFeedsRecall(t *testing.T) {
	ctx := context.Background()
	indexPath := filepath.Join(t.TempDir(), "vectors.json")
	provider := fakeTextEmbeddingProvider{dims: 4}
	index, err := NewLocalVectorIndex(ctx, LocalVectorIndexOptions{Path: indexPath, Provider: provider})
	if err != nil {
		t.Fatalf("NewLocalVectorIndex: %v", err)
	}
	if err := index.Upsert(ctx, LocalVectorMemory{
		MemoryID:    "vec-blue-vault",
		WorkspaceID: "workspace-vector-index",
		Peer:        "peer-vector-index",
		SourceType:  "conclusion",
		Content:     "Maya hid the flower archive in the blue vault.",
		SessionID:   "session-vector-index",
		ScopeID:     MemoryScopeWorkspace,
		Importance:  0.9,
		Metadata:    map[string]string{"source": "fake-test"},
	}); err != nil {
		t.Fatalf("Upsert matching vector: %v", err)
	}
	if err := index.Upsert(ctx, LocalVectorMemory{
		MemoryID:    "vec-other-workspace",
		WorkspaceID: "other-workspace",
		Peer:        "peer-vector-index",
		SourceType:  "conclusion",
		Content:     "Maya hid the flower archive in the red vault.",
		SessionID:   "session-vector-index",
		ScopeID:     MemoryScopeWorkspace,
	}); err != nil {
		t.Fatalf("Upsert other workspace vector: %v", err)
	}

	diag, err := index.Diagnostics(ctx)
	if err != nil {
		t.Fatalf("Diagnostics: %v", err)
	}
	if diag.Path != indexPath || diag.Dimensions != 4 || diag.Count != 2 || diag.Checksum == "" || diag.LastIndexedAt.IsZero() {
		t.Fatalf("diagnostics = %+v, want path/dims/count/checksum/last indexed", diag)
	}

	reopened, err := NewLocalVectorIndex(ctx, LocalVectorIndexOptions{Path: indexPath, Provider: provider})
	if err != nil {
		t.Fatalf("reopen NewLocalVectorIndex: %v", err)
	}
	hits, err := reopened.Search(ctx, VectorSearchQuery{WorkspaceID: "workspace-vector-index", Peer: "peer-vector-index", Query: "flower archive blue vault", SessionKey: "session-vector-index", Limit: 3})
	if err != nil {
		t.Fatalf("Search reopened index: %v", err)
	}
	if len(hits) != 1 || hits[0].MemoryID != "vec-blue-vault" || hits[0].Score <= 0 {
		t.Fatalf("hits = %+v, want persisted matching vector only", hits)
	}
	if hits[0].Metadata["source"] != "fake-test" {
		t.Fatalf("hit metadata = %+v, want cloned metadata", hits[0].Metadata)
	}

	store, err := memory.OpenSqlite(filepath.Join(t.TempDir(), "memory.db"), 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	defer store.Close(ctx)
	if err := RunMigrations(store.DB()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	svc := NewService(store.DB(), Config{WorkspaceID: "workspace-vector-index", ObserverPeerID: "gormes", VectorStore: reopened}, nil)
	trace, err := svc.Recall(ctx, RecallQuery{Peer: "peer-vector-index", Query: "where is the flower archive", SessionKey: "session-vector-index", Limit: 2})
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}
	if !slices.Contains(selectedRecallIDs(trace), "vec-blue-vault") {
		t.Fatalf("selected IDs = %v, want local vector index hit", selectedRecallIDs(trace))
	}
}

type fakeTextEmbeddingProvider struct{ dims int }

func (p fakeTextEmbeddingProvider) EmbedText(_ context.Context, text string) ([]float64, error) {
	return deterministicFakeEmbedding(text, p.dims), nil
}

func deterministicFakeEmbedding(text string, dims int) []float64 {
	if dims <= 0 {
		dims = 4
	}
	out := make([]float64, dims)
	for _, token := range strings.Fields(strings.ToLower(text)) {
		switch strings.Trim(token, ".,;:!?()[]{}\"'") {
		case "flower", "archive":
			out[0]++
		case "blue", "vault":
			out[1]++
		case "red":
			out[2]++
		default:
			out[len(token)%dims] += 0.1
		}
	}
	return out
}

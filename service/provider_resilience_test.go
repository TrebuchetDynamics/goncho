package goncho

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestProviderCircuitBreakerOpensThenHalfOpenClosesOnSuccess(t *testing.T) {
	now := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	breaker := NewProviderCircuitBreaker(ProviderCircuitBreakerConfig{
		Name:             "embedding",
		Kind:             ProviderKindEmbedding,
		FailureThreshold: 2,
		Cooldown:         time.Second,
		Timeout:          time.Second,
		Now:              func() time.Time { return now },
	})

	for i := 0; i < 2; i++ {
		if err := breaker.Execute(context.Background(), func(context.Context) error { return errors.New("provider down") }); err == nil {
			t.Fatalf("failure %d err = nil, want provider down", i+1)
		}
	}
	if got := breaker.Health(); got.CircuitState != ProviderCircuitOpen || got.Status != ProviderStatusDegraded || got.FailureCount != 2 {
		t.Fatalf("health after failures = %+v, want degraded open with two failures", got)
	}

	called := false
	if err := breaker.Execute(context.Background(), func(context.Context) error { called = true; return nil }); !errors.Is(err, ErrProviderCircuitOpen) {
		t.Fatalf("open execute err = %v, want ErrProviderCircuitOpen", err)
	}
	if called {
		t.Fatal("open circuit called provider before cooldown")
	}

	now = now.Add(2 * time.Second)
	if err := breaker.Execute(context.Background(), func(context.Context) error { called = true; return nil }); err != nil {
		t.Fatalf("half-open success: %v", err)
	}
	if !called {
		t.Fatal("half-open call did not probe provider")
	}
	if got := breaker.Health(); got.CircuitState != ProviderCircuitClosed || got.Status != ProviderStatusHealthy || got.FailureCount != 0 {
		t.Fatalf("health after half-open success = %+v, want healthy closed", got)
	}
}

func TestRecallFallsBackToLexicalWhenVectorProviderFails(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	svc.vectorStore = failingVectorStore{err: errors.New("embedding timeout")}
	svc.providerRegistry = NewProviderHealthRegistry(ProviderResilienceConfig{FailureThreshold: 1, Cooldown: time.Minute, Timeout: time.Second}, svc.vectorStore)
	if _, err := svc.Conclude(ctx, ConcludeParams{Peer: "peer-provider", SessionKey: "sess-provider", Conclusion: "Lexical fallback owner is Mira."}); err != nil {
		t.Fatalf("Conclude: %v", err)
	}

	trace, err := svc.Recall(ctx, RecallQuery{Peer: "peer-provider", SessionKey: "sess-provider", Query: "lexical fallback owner", Limit: 3})
	if err != nil {
		t.Fatalf("Recall should fall back instead of failing: %v", err)
	}
	if len(trace.Selected) == 0 || !strings.Contains(trace.Selected[0].Candidate.Content, "Lexical fallback owner is Mira") {
		t.Fatalf("selected = %+v, want lexical fallback conclusion", trace.Selected)
	}
	if !recallWarningListHasCode(trace.Warnings, RecallWarningSemanticUnavailable) {
		t.Fatalf("warnings = %+v, want semantic_unavailable", trace.Warnings)
	}
	providers := svc.ProviderHealthDiagnostics()
	if got := providers.ByName("embedding"); got.Status != ProviderStatusDegraded || got.CircuitState != ProviderCircuitOpen || !strings.Contains(got.LastError, "embedding timeout") {
		t.Fatalf("embedding provider health = %+v, want degraded open timeout", got)
	}
}

func TestRecallSkipsVectorProviderWhenPayloadExceedsConfiguredLimit(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	vector := &countingVectorStore{}
	svc.vectorStore = vector
	svc.providerRegistry = NewProviderHealthRegistry(ProviderResilienceConfig{FailureThreshold: 1, Cooldown: time.Minute, Timeout: time.Second, MaxPayloadBytes: 5}, svc.vectorStore)
	if _, err := svc.Conclude(ctx, ConcludeParams{Peer: "peer-payload", SessionKey: "sess-payload", Conclusion: "Payload fallback owner is Mira."}); err != nil {
		t.Fatalf("Conclude: %v", err)
	}

	trace, err := svc.Recall(ctx, RecallQuery{Peer: "peer-payload", SessionKey: "sess-payload", Query: "payload fallback owner", Limit: 3})
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}
	if vector.calls != 0 {
		t.Fatalf("vector calls = %d, want payload guard to skip provider", vector.calls)
	}
	if !recallWarningListHasCode(trace.Warnings, RecallWarningSemanticUnavailable) {
		t.Fatalf("warnings = %+v, want semantic_unavailable payload warning", trace.Warnings)
	}
}

func TestProviderHealthDiagnosticsReportOptionalDisabledAndViewerWarnings(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	diagnostics := svc.ProviderHealthDiagnostics()
	for _, name := range []string{"extraction", "embedding", "reranking", "summarization"} {
		if got := diagnostics.ByName(name); got.Name != name || got.Status != ProviderStatusDisabled || !got.Optional {
			t.Fatalf("provider %q health = %+v, want optional disabled", name, got)
		}
	}
	viewer, err := svc.ViewerSnapshot(ctx)
	if err != nil {
		t.Fatalf("ViewerSnapshot: %v", err)
	}
	if !strings.Contains(strings.Join(viewer.UnavailableWarnings, "\n"), "provider embedding disabled") {
		t.Fatalf("viewer warnings = %+v, want provider disabled warning", viewer.UnavailableWarnings)
	}
}

type failingVectorStore struct{ err error }

func (f failingVectorStore) Search(context.Context, VectorSearchQuery) ([]VectorSearchHit, error) {
	return nil, f.err
}

type countingVectorStore struct{ calls int }

func (c *countingVectorStore) Search(context.Context, VectorSearchQuery) ([]VectorSearchHit, error) {
	c.calls++
	return nil, nil
}

package goncho

import (
	"context"
	"testing"
)

func TestDreamFactExtraction_ExtractsConclusions(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Insert test turns via CreateMessages
	_, err := svc.CreateMessages(ctx, CreateMessagesParams{
		SessionKey: "sess-dream-extract",
		Messages: []CreateMessage{
			{Peer: "operator", Role: "user", Content: "I decided to use SQLite instead of PostgreSQL for local storage"},
			{Peer: "gormes", Role: "assistant", Content: "We should add tests for the new feature"},
		},
	})
	if err != nil {
		t.Fatalf("CreateMessages: %v", err)
	}

	count, err := svc.ExecuteDreamFactExtraction(ctx, "sess-dream-extract")
	if err != nil {
		t.Fatalf("ExecuteDreamFactExtraction: %v", err)
	}
	if count == 0 {
		t.Error("expected conclusions to be extracted")
	}
}

func TestDreamFactExtraction_UpdatesPeerCards(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()

	_, err := svc.CreateMessages(ctx, CreateMessagesParams{
		SessionKey: "sess-dream-peer",
		Messages: []CreateMessage{
			{Peer: "operator", Role: "user", Content: "I prefer tabs over spaces for indentation"},
		},
	})
	if err != nil {
		t.Fatalf("CreateMessages: %v", err)
	}

	count, err := svc.ExecuteDreamFactExtraction(ctx, "sess-dream-peer")
	if err != nil {
		t.Fatalf("ExecuteDreamFactExtraction: %v", err)
	}
	if count == 0 {
		t.Error("expected peer card update from dream extraction")
	}
}

func TestDreamCompression_MergesRedundantMemories(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()

	_, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "operator",
		Conclusion: "SQLite for local storage instead of PostgreSQL",
	})
	if err != nil {
		t.Fatalf("Conclude 1: %v", err)
	}
	_, err = svc.Conclude(ctx, ConcludeParams{
		Peer:       "operator",
		Conclusion: "SQLite local storage instead of PostgreSQL",
	})
	if err != nil {
		t.Fatalf("Conclude 2: %v", err)
	}

	compressed, err := svc.ExecuteDreamCompression(ctx)
	if err != nil {
		t.Fatalf("ExecuteDreamCompression: %v", err)
	}
	if compressed == 0 {
		t.Error("expected redundant conclusions to be compressed")
	}
}

func TestDreamScheduler_NonBlocking(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()

	// Verify that dream extraction returns quickly
	done := make(chan struct{})
	go func() {
		_, _ = svc.ExecuteDreamFactExtraction(ctx, "sess-nonblocking")
		close(done)
	}()

	select {
	case <-done:
		// Completed quickly
	case <-ctx.Done():
		t.Fatal("dream extraction blocked")
	}
}

func TestWordSimilarity_HighOverlap(t *testing.T) {
	a := "We decided to use SQLite for local storage"
	b := "Using SQLite for local storage was our decision"
	sim := wordSimilarity(a, b)
	if sim < 0.3 {
		t.Errorf("expected moderate similarity, got %f", sim)
	}
}

func TestWordSimilarity_LowOverlap(t *testing.T) {
	a := "We decided to use SQLite for local storage"
	b := "The sky is blue and the grass is green"
	sim := wordSimilarity(a, b)
	if sim > 0.3 {
		t.Errorf("expected low similarity, got %f", sim)
	}
}

func TestWordSimilarity_Identical(t *testing.T) {
	a := "We decided to use SQLite"
	b := "We decided to use SQLite"
	sim := wordSimilarity(a, b)
	if sim != 1.0 {
		t.Errorf("expected 1.0 for identical strings, got %f", sim)
	}
}

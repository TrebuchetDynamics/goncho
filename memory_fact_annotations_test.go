package goncho

import (
	"context"
	"testing"
)

func TestRunMigrationsCreatesMemoryAnnotationTableIdempotently(t *testing.T) {
	db := openObservationTestDB(t)
	ctx := context.Background()

	if err := RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations first: %v", err)
	}
	if err := RunMigrations(db); err != nil {
		t.Fatalf("RunMigrations second: %v", err)
	}
	if !observationTableExists(ctx, t, db, "goncho_memory_annotations") {
		t.Fatal("goncho_memory_annotations table does not exist")
	}
}

func TestServiceConcludeFactAnnotationsRankInverseOwnerFact(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	answer, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Project note: component A-17's owner is Nadia.",
		SessionKey: "sess-annotation-fact",
	})
	if err != nil {
		t.Fatal(err)
	}
	decoy, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Who owns component A-17? owns component owns component owns component owns component. This checklist repeats the retrieval words but does not answer it.",
		SessionKey: "sess-annotation-fact",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.db.ExecContext(ctx, `
		UPDATE goncho_conclusions
		SET updated_at = CASE id WHEN ? THEN 200 WHEN ? THEN 100 ELSE updated_at END
		WHERE id IN (?, ?)
	`, decoy.ID, answer.ID, decoy.ID, answer.ID); err != nil {
		t.Fatalf("force decoy recency: %v", err)
	}

	got, err := svc.Search(ctx, SearchParams{
		Peer:       "team",
		Query:      "Who owns component A-17?",
		SessionKey: "sess-annotation-fact",
		Limit:      2,
		MaxTokens:  200,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Results) < 2 {
		t.Fatalf("results = %+v, want answer and decoy", got.Results)
	}
	if got.Results[0].ID != answer.ID {
		t.Fatalf("first result = %+v, want inverse owner fact before lexical echo", got.Results[0])
	}
}

package goncho

import (
	"context"
	"strings"
	"testing"
)

func TestServiceConcludeSequenceAnnotationRanksDurableSequence(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	answer, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Project note: Release rollout sequence: first freeze writes, then run migration, finally enable readers.",
		SessionKey: "sess-sequence-fact",
	})
	if err != nil {
		t.Fatal(err)
	}
	decoy, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Walk me through the release rollout sequence? release rollout sequence release rollout sequence release rollout sequence. This checklist repeats the retrieval words but does not state the order.",
		SessionKey: "sess-sequence-fact",
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

	var annotation string
	if err := svc.db.QueryRowContext(ctx, `
		SELECT value
		FROM goncho_memory_annotations
		WHERE memory_source = 'conclusion'
		  AND memory_id = ?
		  AND kind = 'fact'
		ORDER BY id
		LIMIT 1
	`, answer.ID).Scan(&annotation); err != nil {
		t.Fatalf("lookup sequence annotation: %v", err)
	}
	for _, want := range []string{"Release rollout sequence", "first freeze writes", "then run migration", "finally enable readers"} {
		if !strings.Contains(annotation, want) {
			t.Fatalf("annotation = %q, want durable sequence fact containing %q", annotation, want)
		}
	}

	got, err := svc.Search(ctx, SearchParams{
		Peer:       "team",
		Query:      "Walk me through the release rollout sequence.",
		SessionKey: "sess-sequence-fact",
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
		t.Fatalf("first result = %+v, want durable sequence fact before lexical echo", got.Results[0])
	}
}

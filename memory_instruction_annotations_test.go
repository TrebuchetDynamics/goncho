package goncho

import (
	"context"
	"strings"
	"testing"
)

func TestServiceConcludeInstructionAnnotationRanksDurableInstruction(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	answer, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Project note: Mira's instruction is never delete logs.",
		SessionKey: "sess-instruction-fact",
	})
	if err != nil {
		t.Fatal(err)
	}
	decoy, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "What instruction did Mira give about logs? instruction logs instruction logs instruction logs instruction logs. This checklist repeats the retrieval words but does not state the rule.",
		SessionKey: "sess-instruction-fact",
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
		t.Fatalf("lookup instruction annotation: %v", err)
	}
	if !strings.Contains(annotation, "Mira instructed") || !strings.Contains(annotation, "delete logs") {
		t.Fatalf("annotation = %q, want durable instruction fact", annotation)
	}

	got, err := svc.Search(ctx, SearchParams{
		Peer:       "team",
		Query:      "What instruction did Mira give about logs?",
		SessionKey: "sess-instruction-fact",
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
		t.Fatalf("first result = %+v, want durable instruction fact before lexical echo", got.Results[0])
	}
}

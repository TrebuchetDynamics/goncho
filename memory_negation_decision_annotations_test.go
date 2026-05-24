package goncho

import (
	"context"
	"strings"
	"testing"
)

func TestServiceConcludeNegationAnnotationRanksDurableDenial(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	answer, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Project note: I never approved auto-deleting audit logs.",
		SessionKey: "sess-negation-fact",
	})
	if err != nil {
		t.Fatal(err)
	}
	decoy, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Have I approved auto-deleting audit logs? approved auto-deleting audit logs approved auto-deleting audit logs. This checklist repeats the retrieval words but does not state the denial.",
		SessionKey: "sess-negation-fact",
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
		t.Fatalf("lookup negation annotation: %v", err)
	}
	if !strings.Contains(annotation, "user never approved auto-deleting audit logs") {
		t.Fatalf("annotation = %q, want durable negation fact", annotation)
	}

	got, err := svc.Search(ctx, SearchParams{
		Peer:       "team",
		Query:      "Have I approved auto-deleting audit logs?",
		SessionKey: "sess-negation-fact",
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
		t.Fatalf("first result = %+v, want durable negation fact before lexical echo", got.Results[0])
	}
}

func TestServiceConcludeDecisionAnnotationRanksDurableDecision(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	answer, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Project note: I decided to keep PostgreSQL for audit logs.",
		SessionKey: "sess-decision-fact",
	})
	if err != nil {
		t.Fatal(err)
	}
	decoy, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "What decision did I make about audit logs? decision audit logs decision audit logs decision audit logs. This checklist repeats the retrieval words but does not state the decision.",
		SessionKey: "sess-decision-fact",
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
		t.Fatalf("lookup decision annotation: %v", err)
	}
	if !strings.Contains(annotation, "user decided to keep PostgreSQL for audit logs") {
		t.Fatalf("annotation = %q, want durable decision fact", annotation)
	}

	got, err := svc.Search(ctx, SearchParams{
		Peer:       "team",
		Query:      "What decision did I make about audit logs?",
		SessionKey: "sess-decision-fact",
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
		t.Fatalf("first result = %+v, want durable decision fact before lexical echo", got.Results[0])
	}
}

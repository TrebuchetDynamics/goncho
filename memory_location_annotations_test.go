package goncho

import (
	"context"
	"strings"
	"testing"
)

func TestServiceConcludeLocationAnnotationRanksDurableLocation(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	answer, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Project note: the escalation runbook location is Notion page RB-17.",
		SessionKey: "sess-location-fact",
	})
	if err != nil {
		t.Fatal(err)
	}
	decoy, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Where is the escalation runbook? escalation runbook escalation runbook escalation runbook. This checklist repeats the retrieval words but does not answer it.",
		SessionKey: "sess-location-fact",
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
		t.Fatalf("lookup location annotation: %v", err)
	}
	if !strings.Contains(annotation, "escalation runbook") || !strings.Contains(annotation, "Notion page RB-17") {
		t.Fatalf("annotation = %q, want durable location fact", annotation)
	}

	got, err := svc.Search(ctx, SearchParams{
		Peer:       "team",
		Query:      "Where is the escalation runbook?",
		SessionKey: "sess-location-fact",
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
		t.Fatalf("first result = %+v, want durable location fact before lexical echo", got.Results[0])
	}
}

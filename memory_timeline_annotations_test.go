package goncho

import (
	"context"
	"strings"
	"testing"
)

func TestServiceConcludeTimelineAnnotationRanksDurableDate(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	answer, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Project note: Release Orion deadline is 2026-06-01.",
		SessionKey: "sess-timeline-fact",
	})
	if err != nil {
		t.Fatal(err)
	}
	decoy, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "When is Release Orion? Release Orion when Release Orion when Release Orion when. This checklist repeats the retrieval words but does not state the date.",
		SessionKey: "sess-timeline-fact",
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
		t.Fatalf("lookup timeline annotation: %v", err)
	}
	if !strings.Contains(annotation, "Release Orion") || !strings.Contains(annotation, "2026-06-01") {
		t.Fatalf("annotation = %q, want durable timeline fact", annotation)
	}

	got, err := svc.Search(ctx, SearchParams{
		Peer:       "team",
		Query:      "When is Release Orion?",
		SessionKey: "sess-timeline-fact",
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
		t.Fatalf("first result = %+v, want durable timeline fact before lexical echo", got.Results[0])
	}
}

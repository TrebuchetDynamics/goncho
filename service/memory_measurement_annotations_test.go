package goncho

import (
	"context"
	"strings"
	"testing"
)

func TestServiceConcludeMetricAnnotationRanksDurableMetric(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	answer, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Project note: dashboard API latency is 250ms.",
		SessionKey: "sess-metric-fact",
	})
	if err != nil {
		t.Fatal(err)
	}
	decoy, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "What is the dashboard API latency? dashboard API latency dashboard API latency dashboard API latency. This checklist repeats the retrieval words but does not state the metric.",
		SessionKey: "sess-metric-fact",
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
		t.Fatalf("lookup metric annotation: %v", err)
	}
	if !strings.Contains(annotation, "dashboard API latency") || !strings.Contains(annotation, "250ms") {
		t.Fatalf("annotation = %q, want durable metric fact", annotation)
	}

	got, err := svc.Search(ctx, SearchParams{
		Peer:       "team",
		Query:      "What is the dashboard API latency?",
		SessionKey: "sess-metric-fact",
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
		t.Fatalf("first result = %+v, want durable metric fact before lexical echo", got.Results[0])
	}
}

func TestServiceConcludeVersionAnnotationRanksDurableVersion(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	answer, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Project note: PostgreSQL version is 14.2.",
		SessionKey: "sess-version-fact",
	})
	if err != nil {
		t.Fatal(err)
	}
	decoy, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "What PostgreSQL version? PostgreSQL version PostgreSQL version PostgreSQL version. This checklist repeats the retrieval words but does not state the version.",
		SessionKey: "sess-version-fact",
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
		t.Fatalf("lookup version annotation: %v", err)
	}
	if !strings.Contains(annotation, "PostgreSQL version") || !strings.Contains(annotation, "14.2") {
		t.Fatalf("annotation = %q, want durable version fact", annotation)
	}

	got, err := svc.Search(ctx, SearchParams{
		Peer:       "team",
		Query:      "What PostgreSQL version?",
		SessionKey: "sess-version-fact",
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
		t.Fatalf("first result = %+v, want durable version fact before lexical echo", got.Results[0])
	}
}

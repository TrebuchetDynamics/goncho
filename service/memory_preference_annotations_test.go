package goncho

import (
	"context"
	"testing"
)

func TestServiceConcludePreferenceAnnotationRanksDurablePreference(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	answer, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Project note: Mira's indentation preference is tabs.",
		SessionKey: "sess-preference-fact",
	})
	if err != nil {
		t.Fatal(err)
	}
	decoy, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "What indentation does Mira prefer? indentation prefer indentation prefer indentation prefer. This checklist repeats the retrieval words but does not answer it.",
		SessionKey: "sess-preference-fact",
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
		Query:      "What indentation does Mira prefer?",
		SessionKey: "sess-preference-fact",
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
		t.Fatalf("first result = %+v, want durable preference fact before lexical echo", got.Results[0])
	}
}

package goncho

import (
	"context"
	"strings"
	"testing"
)

func TestSearchFactIntentScoresStructuredSpeakerAnnotationOnlyForSpeakerQuestions(t *testing.T) {
	if score := searchFactIntentScore("What did Melanie say about the sunrise?", "speaker Melanie"); score != 1 {
		t.Fatalf("speaker fact score = %v, want 1", score)
	}
	if score := searchFactIntentScore("When did Melanie paint a sunrise?", "speaker Melanie"); score != 0 {
		t.Fatalf("non-speaker question fact score = %v, want 0", score)
	}
	if score := searchFactIntentScore("What did Caroline say about the sunrise?", "speaker Melanie"); score != 0 {
		t.Fatalf("mismatched speaker fact score = %v, want 0", score)
	}
}

func TestSearchFactIntentScoresOwnerAnswerButNotLexicalEcho(t *testing.T) {
	query := "Who owns component A-17?"
	if score := searchFactIntentScore(query, "Nadia owns component A-17."); score != 1 {
		t.Fatalf("answer fact score = %v, want 1", score)
	}
	if score := searchFactIntentScore(query, "Who owns component A-17? owns component owns component owns component owns component. This checklist repeats the retrieval words but does not answer it."); score != 0 {
		t.Fatalf("lexical echo score = %v, want 0", score)
	}
}

func TestServiceSearchFactIntentRanksAnswerOverLexicalEcho(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()

	ctx := context.Background()
	answer, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Nadia owns component A-17.",
		SessionKey: "sess-fact-intent",
	})
	if err != nil {
		t.Fatal(err)
	}
	decoy, err := svc.Conclude(ctx, ConcludeParams{
		Peer:       "team",
		Conclusion: "Who owns component A-17? owns component owns component owns component owns component. This checklist repeats the retrieval words but does not answer it.",
		SessionKey: "sess-fact-intent",
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
		SessionKey: "sess-fact-intent",
		Limit:      2,
		MaxTokens:  200,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Results) < 2 {
		t.Fatalf("results = %+v, want answer and decoy", got.Results)
	}
	if got.Results[0].ID != answer.ID || !strings.Contains(got.Results[0].Content, "Nadia owns component A-17") {
		t.Fatalf("first result = %+v, want answer fact before lexical echo", got.Results[0])
	}
}

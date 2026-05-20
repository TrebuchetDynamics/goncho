package goncho

import "testing"

func TestSearchRankTemporalIntentExtraction(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  searchTemporalDirection
	}{
		{name: "latest", query: "Which streaming service did I start using most recently?", want: searchTemporalNewer},
		{name: "current", query: "How many projects am I currently leading?", want: searchTemporalNewer},
		{name: "first", query: "Which event did I attend first?", want: searchTemporalOlder},
		{name: "initial", query: "How many plants did I initially plant?", want: searchTemporalOlder},
		{name: "none", query: "What action figure did I buy?", want: searchTemporalNone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := searchTemporalIntent(tt.query).Direction; got != tt.want {
				t.Fatalf("direction = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSearchRankTemporalMarkers(t *testing.T) {
	features := searchTemporalIntent("What is the order of the sports events I watched in January?")
	if len(features.Markers) != 1 || features.Markers[0] != "january" {
		t.Fatalf("markers = %#v, want january", features.Markers)
	}
}

func TestRankConclusionHitsTemporalDoesNotPenalizePersonalMemoryWithGenericAside(t *testing.T) {
	hits := []SearchHit{
		{Content: "user: What shampoo should I buy? assistant: I don't have personal experience, but here is a generic list of shampoo brands."},
		{Content: "user: I'm organizing my bathroom. My shampoo is Acme. My routine uses my Acme shampoo. My bathroom shelf has my shampoo bottle. My current shampoo brand is Acme. assistant: Your current shampoo is Acme."},
	}
	got := rankConclusionHitsByLexicalOverlap("What brand of shampoo do I currently use?", hits)
	if got[0].Content != hits[1].Content {
		t.Fatalf("top = %q, want personal memory with generic aside preserved", got[0].Content)
	}
}

func TestRankConclusionHitsTemporalPenalizesGenericAIBoilerplate(t *testing.T) {
	hits := []SearchHit{
		{Content: "user: How many weddings happen this year? assistant: As an AI language model, I don't have personal experience with weddings, but here is general information."},
		{Content: "user: I attended my roommate's wedding this year and also my cousin's wedding this year. assistant: That is two weddings this year."},
	}
	got := rankConclusionHitsByLexicalOverlap("How many weddings have I attended in this year?", hits)
	if got[0].Content != hits[1].Content {
		t.Fatalf("top = %q, want personal temporal memory before generic AI boilerplate", got[0].Content)
	}
}

func TestRankConclusionHitsTemporalRerankOnlyNearTies(t *testing.T) {
	hits := []SearchHit{
		{Content: "user: I tried an old music streaming service last year. assistant: That older service is fine."},
		{Content: "user: I started using a new music streaming service most recently. assistant: The recent streaming choice fits."},
	}
	got := rankConclusionHitsByLexicalOverlap("Which streaming service did I start using most recently?", hits)
	if got[0].Content != hits[1].Content {
		t.Fatalf("top = %q, want newer temporal hit", got[0].Content)
	}

	strongLexical := []SearchHit{
		{Content: "user: Spotify Spotify Spotify streaming service most recently. assistant: Spotify was the answer."},
		{Content: "user: I used a music app recently. assistant: A recent app."},
	}
	got = rankConclusionHitsByLexicalOverlap("Which streaming service did I start using most recently?", strongLexical)
	if got[0].Content != strongLexical[0].Content {
		t.Fatalf("top = %q, want strong lexical evidence preserved", got[0].Content)
	}
}

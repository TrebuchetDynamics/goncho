package driftanchor

import "testing"

func TestSimilarityWarnsOnSharedSpecificTokens(t *testing.T) {
	prompt := "Let's retry the stale Docker cache cleanup again before checking container state."
	memory := "Dead end: retrying stale Docker cache cleanup repeats a known failure; verify live container state first."
	if got := Similarity(prompt, memory); got < 0.30 {
		t.Fatalf("Similarity() = %v, want >= 0.30", got)
	}
}

func TestSimilarityIgnoresGenericStopwords(t *testing.T) {
	prompt := "this should again happen before that"
	memory := "known from after with could would"
	if got := Similarity(prompt, memory); got != 0 {
		t.Fatalf("Similarity() = %v, want 0", got)
	}
}

func TestTokenSetDropsShortTokensAndStopwords(t *testing.T) {
	got := TokenSet("AI stale Docker before cleanup")
	want := map[string]struct{}{"stale": {}, "docker": {}, "cleanup": {}}
	if len(got) != len(want) {
		t.Fatalf("TokenSet() = %#v, want %#v", got, want)
	}
	for token := range want {
		if _, ok := got[token]; !ok {
			t.Fatalf("TokenSet() = %#v, missing %q", got, token)
		}
	}
}

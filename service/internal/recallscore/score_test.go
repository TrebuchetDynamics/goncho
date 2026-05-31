package recallscore

import "testing"

func TestKeywordScoresExactPhrase(t *testing.T) {
	got := Keyword("Maya stores the rare orchid retrieval marker in the archive cabinet.", "rare orchid retrieval marker")
	if got != 1 {
		t.Fatalf("Keyword exact phrase = %v, want 1", got)
	}
}

func TestKeywordTokenOverlapUsesWholeTokens(t *testing.T) {
	got := Keyword("Maya keeps the shopping list in the kitchen.", "pin")
	if got != 0 {
		t.Fatalf("Keyword substring-only token overlap = %v, want 0", got)
	}
}

func TestKeywordTokenOverlapNormalizesPunctuation(t *testing.T) {
	got := Keyword("Maya's archive-cabinet stores the marker.", "archive cabinet")
	if got != 1 {
		t.Fatalf("Keyword punctuation-normalized overlap = %v, want 1", got)
	}
}

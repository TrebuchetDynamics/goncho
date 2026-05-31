package benchmarkscore

import "testing"

func TestRecallAtKCountsUniqueRelevantHits(t *testing.T) {
	got := RecallAtK([]string{"a", "b", "b", "c"}, []string{"b", "c"}, 3)
	if got != 0.5 {
		t.Fatalf("RecallAtK = %v, want 0.5", got)
	}
}

func TestRubricCoverageMatchesSignificantTokensAcrossContexts(t *testing.T) {
	score, matches := RubricCoverage(
		[]string{"Mira owns the lexical fallback owner and provider warnings."},
		[]string{"identify lexical fallback owner", "mention provider warnings", "state budget policy"},
	)
	if score != 0.666667 {
		t.Fatalf("score = %v, want 0.666667", score)
	}
	if len(matches) != 2 || matches[0] != "identify lexical fallback owner" || matches[1] != "mention provider warnings" {
		t.Fatalf("matches = %+v, want first two rubric items", matches)
	}
}

func TestRubricTokensDropsStopWordsAndShortTokens(t *testing.T) {
	got := RubricTokens("Correctly identifies: A/B project owner!")
	want := []string{"owner"}
	if len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("RubricTokens = %+v, want %+v", got, want)
	}
}

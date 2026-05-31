package textmatch

import "testing"

func TestCoverageAndJaccard(t *testing.T) {
	a := map[string]struct{}{"sqlite": {}, "local": {}, "storage": {}}
	b := map[string]struct{}{"sqlite": {}, "storage": {}, "postgres": {}}

	if got, want := IntersectionSize(a, b), 2; got != want {
		t.Fatalf("IntersectionSize() = %v, want %v", got, want)
	}
	if got, want := Coverage(a, b), 2.0/3.0; got != want {
		t.Fatalf("Coverage() = %v, want %v", got, want)
	}
	if got, want := Jaccard(a, b), 0.5; got != want {
		t.Fatalf("Jaccard() = %v, want %v", got, want)
	}
}

func TestJaccardSignificantWords(t *testing.T) {
	if got := JaccardSignificantWords("We decided to use SQLite", "We decided to use SQLite"); got != 1 {
		t.Fatalf("JaccardSignificantWords identical = %v, want 1", got)
	}
	if got := JaccardSignificantWords("We decided to use SQLite for local storage", "Using SQLite for local storage was our decision"); got < 0.3 {
		t.Fatalf("JaccardSignificantWords overlap = %v, want >= 0.3", got)
	}
	if got := JaccardSignificantWords("We decided to use SQLite for local storage", "The sky is blue and the grass is green"); got > 0.3 {
		t.Fatalf("JaccardSignificantWords low overlap = %v, want <= 0.3", got)
	}
}

func TestSignificantWordSetDropsShortTokensAndPunctuation(t *testing.T) {
	got := SignificantWordSet("AI in SQLite, local-storage!")
	want := map[string]struct{}{"sqlite": {}, "local-storage": {}}
	if len(got) != len(want) {
		t.Fatalf("SignificantWordSet() = %#v, want %#v", got, want)
	}
	for token := range want {
		if _, ok := got[token]; !ok {
			t.Fatalf("SignificantWordSet() = %#v, missing %q", got, token)
		}
	}
}

func TestOverlapCoefficient(t *testing.T) {
	a := map[string]struct{}{"repeat": {}, "failed": {}, "path": {}}
	b := map[string]struct{}{"failed": {}, "path": {}, "again": {}, "later": {}}

	got := OverlapCoefficient(a, b)
	want := 2.0 / 3.0
	if got != want {
		t.Fatalf("OverlapCoefficient() = %v, want %v", got, want)
	}
}

func TestSimilarityEmptySets(t *testing.T) {
	if got := Jaccard(nil, map[string]struct{}{"x": {}}); got != 0 {
		t.Fatalf("Jaccard empty = %v, want 0", got)
	}
	if got := OverlapCoefficient(map[string]struct{}{"x": {}}, nil); got != 0 {
		t.Fatalf("OverlapCoefficient empty = %v, want 0", got)
	}
}

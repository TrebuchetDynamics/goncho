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

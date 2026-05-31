package scoring

import (
	"math"
	"testing"
)

func TestBM25ReturnsZeroForIncompleteCorpusStats(t *testing.T) {
	queryTokens := map[string]struct{}{"orchid": {}}
	tf := map[string]int{"orchid": 2}
	df := map[string]int{"orchid": 0}

	got := BM25(queryTokens, tf, df, 3, 2, 2)
	if got != 0 {
		t.Fatalf("BM25 with missing document frequency = %v, want 0", got)
	}
}

func TestBM25ClampsImpossibleDocumentFrequencyToCorpusSize(t *testing.T) {
	queryTokens := map[string]struct{}{"orchid": {}}
	tf := map[string]int{"orchid": 1}
	df := map[string]int{"orchid": 4}

	got := BM25(queryTokens, tf, df, 2, 1, 1)
	if got < 0 || math.IsNaN(got) || math.IsInf(got, 0) {
		t.Fatalf("BM25 with df > docCount = %v, want finite non-negative score", got)
	}
}

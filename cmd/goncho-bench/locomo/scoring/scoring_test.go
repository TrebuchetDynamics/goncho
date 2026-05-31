package scoring

import "testing"

func TestScoringStrictAnyAndNDCG(t *testing.T) {
	retrieved := []string{"m1", "m2", "m3", "m4"}
	gold := []string{"m2", "m4"}
	if RecallAny(retrieved, gold, 5) != 1 || StrictRecall(retrieved, gold, 5) != 1 {
		t.Fatalf("@5 any/strict = %.4f/%.4f, want 1/1", RecallAny(retrieved, gold, 5), StrictRecall(retrieved, gold, 5))
	}
	if got := NDCG(retrieved, gold, 5); got != 0.6509 {
		t.Fatalf("ndcg@5 = %.4f, want 0.6509", got)
	}
	if StrictRecall([]string{"m1", "m2", "m3"}, gold, 5) != 0 {
		t.Fatal("strict recall should require all gold IDs")
	}
}

func TestScoringHandlesEmptyAndNonPositiveWindows(t *testing.T) {
	if RecallAny([]string{"m1"}, []string{"m1"}, 0) != 0 {
		t.Fatal("recall-any should be 0 for top-0 window")
	}
	if StrictRecall([]string{"m1"}, []string{"m1"}, 0) != 0 {
		t.Fatal("strict recall should be 0 for top-0 window with non-empty gold")
	}
	if NDCG([]string{"m1"}, []string{"m1"}, 0) != 0 || NDCG([]string{"m1"}, nil, 5) != 0 {
		t.Fatal("ndcg should be 0 for top-0 or empty gold")
	}
}

package ranking

import "testing"

func TestFirstRelevantRankReturnsOneBasedFirstHit(t *testing.T) {
	if got := FirstRelevantRank([]string{"m0", "m2", "m1"}, []string{"m1", "m2"}); got != 2 {
		t.Fatalf("FirstRelevantRank = %d, want 2", got)
	}
	if got := FirstRelevantRank([]string{"m0"}, []string{"m1"}); got != 0 {
		t.Fatalf("FirstRelevantRank miss = %d, want 0", got)
	}
}

func TestRecallAtKDeduplicatesRelevantHits(t *testing.T) {
	got := RecallAtK([]string{"m1", "m1", "m2", "m3"}, []string{"m1", "m2", "m4"}, 3)
	if got != 0.6667 {
		t.Fatalf("RecallAtK = %v, want 0.6667", got)
	}
}

func TestStringSetTrimsEmptyValues(t *testing.T) {
	set := StringSet([]string{" a ", "", "b"})
	if _, ok := set["a"]; !ok || len(set) != 2 {
		t.Fatalf("StringSet = %#v, want trimmed non-empty IDs", set)
	}
}

func TestTopNReturnsCopiedPrefix(t *testing.T) {
	values := []string{"a", "b", "c"}
	got := TopN(values, 2)
	got[0] = "changed"
	if values[0] != "a" || len(got) != 2 {
		t.Fatalf("TopN got=%v source=%v, want copied first two values", got, values)
	}
}

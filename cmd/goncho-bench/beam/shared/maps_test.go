package shared

import "testing"

func TestSortedStringMapKeysReturnsStableOrder(t *testing.T) {
	got := SortedStringMapKeys(map[string]int{"charlie": 3, "alpha": 1, "bravo": 2})
	want := []string{"alpha", "bravo", "charlie"}
	if len(got) != len(want) {
		t.Fatalf("SortedStringMapKeys() len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SortedStringMapKeys()[%d] = %q, want %q (all: %v)", i, got[i], want[i], got)
		}
	}
}

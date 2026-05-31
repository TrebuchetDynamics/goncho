package sliceutil

import "testing"

func TestIndexByKeepsFirstAcceptedIndex(t *testing.T) {
	values := []string{"alpha", "", "beta", "alpha"}
	got := IndexBy(values, func(value string) (string, bool) {
		return value, value != ""
	})

	want := map[string]int{"alpha": 0, "beta": 2}
	if len(got) != len(want) {
		t.Fatalf("IndexBy() len = %d, want %d: %#v", len(got), len(want), got)
	}
	for key, wantIndex := range want {
		if got[key] != wantIndex {
			t.Fatalf("IndexBy()[%q] = %d, want %d", key, got[key], wantIndex)
		}
	}
	if _, exists := got[""]; exists {
		t.Fatalf("IndexBy() included rejected blank key: %#v", got)
	}
}

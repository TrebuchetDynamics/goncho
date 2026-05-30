package sliceutil

import "testing"

func TestMapPreservesNilInput(t *testing.T) {
	if got := Map[string, int](nil, func(value string) int { return len(value) }); got != nil {
		t.Fatalf("Map(nil) = %#v, want nil", got)
	}
}

func TestMapAppliesFunctionInOrder(t *testing.T) {
	got := Map([]string{"alpha", "go"}, func(value string) int { return len(value) })
	want := []int{5, 2}
	if len(got) != len(want) {
		t.Fatalf("Map length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Map()[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestMapNilFunctionReturnsZeroValues(t *testing.T) {
	got := Map[string, int]([]string{"alpha", "go"}, nil)
	want := []int{0, 0}
	if len(got) != len(want) {
		t.Fatalf("Map length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Map()[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

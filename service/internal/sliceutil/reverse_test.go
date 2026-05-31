package sliceutil

import "testing"

func TestReverseCloneReturnsReversedCopy(t *testing.T) {
	input := []string{"first", "second", "third"}
	got := ReverseClone(input)
	want := []string{"third", "second", "first"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	got[0] = "changed"
	if input[2] != "third" {
		t.Fatalf("ReverseClone aliased input: input[2] = %q", input[2])
	}
}

func TestReverseCloneEmpty(t *testing.T) {
	got := ReverseClone([]int{})
	if got == nil {
		t.Fatalf("ReverseClone(empty) returned nil, want non-nil empty slice")
	}
	if len(got) != 0 {
		t.Fatalf("len = %d, want 0", len(got))
	}
}

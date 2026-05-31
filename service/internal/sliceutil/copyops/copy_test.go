package copyops

import "testing"

func TestClonePreservesNilInput(t *testing.T) {
	if got := Clone[int](nil); got != nil {
		t.Fatalf("Clone(nil) = %#v, want nil", got)
	}
}

func TestCloneReturnsIndependentCopy(t *testing.T) {
	input := []int{1, 2, 3}
	got := Clone(input)
	got[0] = 99
	if input[0] != 1 {
		t.Fatalf("Clone aliased input: input = %#v", input)
	}
}

func TestReverseCloneReturnsReversedIndependentCopy(t *testing.T) {
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

func TestReverseCloneEmptyIsNonNil(t *testing.T) {
	got := ReverseClone([]int{})
	if got == nil {
		t.Fatalf("ReverseClone(empty) returned nil, want non-nil empty slice")
	}
}

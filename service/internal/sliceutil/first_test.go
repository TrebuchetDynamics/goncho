package sliceutil

import "testing"

func TestFirst(t *testing.T) {
	if got := First([]string{"a", "b"}); got != "a" {
		t.Fatalf("First(strings) = %q, want a", got)
	}
	if got := First([]int(nil)); got != 0 {
		t.Fatalf("First(empty ints) = %d, want zero", got)
	}
}

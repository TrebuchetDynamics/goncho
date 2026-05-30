package textutil

import "testing"

func TestHasAnyPrefixFold(t *testing.T) {
	t.Parallel()

	if !HasAnyPrefixFold("Next: ship it", "todo:", "next:") {
		t.Fatal("expected folded prefix match")
	}
	if HasAnyPrefixFold("ship it", "", "todo:") {
		t.Fatal("expected no prefix match")
	}
}

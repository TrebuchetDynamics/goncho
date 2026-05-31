package stableid

import "testing"

func TestTrimmedNullSeparatedIsStableAndTrimsParts(t *testing.T) {
	got := TrimmedNullSeparated(12, " eval ", "workspace", "question")
	want := TrimmedNullSeparated(12, "eval", "workspace", "question")
	if got != want {
		t.Fatalf("TrimmedNullSeparated did not trim parts: %q != %q", got, want)
	}
	if len(got) != 24 {
		t.Fatalf("TrimmedNullSeparated length = %d, want 24 hex chars", len(got))
	}
}

func TestTrimmedNullSeparatedSeparatesAdjacentFields(t *testing.T) {
	left := TrimmedNullSeparated(12, "ab", "c")
	right := TrimmedNullSeparated(12, "a", "bc")
	if left == right {
		t.Fatalf("TrimmedNullSeparated collided for adjacent fields: %q", left)
	}
}

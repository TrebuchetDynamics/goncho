package shared

import "testing"

func TestFirstNonEmptyTrimmedReturnsFirstTrimmedValue(t *testing.T) {
	if got := FirstNonEmptyTrimmed("", "  ", " candidate ", "fallback"); got != "candidate" {
		t.Fatalf("FirstNonEmptyTrimmed() = %q, want candidate", got)
	}
	if got := FirstNonEmptyTrimmed("", "  "); got != "" {
		t.Fatalf("FirstNonEmptyTrimmed() with blanks = %q, want empty", got)
	}
}

func TestHasNonEmptyTrimmed(t *testing.T) {
	if HasNonEmptyTrimmed("  ") {
		t.Fatal("HasNonEmptyTrimmed(blank) = true, want false")
	}
	if !HasNonEmptyTrimmed(" value ") {
		t.Fatal("HasNonEmptyTrimmed(value) = false, want true")
	}
}

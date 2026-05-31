package wordspace

import (
	"testing"
	"unicode/utf8"
)

func TestCollapseFirstWordsAndCompactContracts(t *testing.T) {
	words := Words("  alpha\n\tbeta   gamma  ")
	if len(words) != 3 || words[0] != "alpha" || words[1] != "beta" || words[2] != "gamma" {
		t.Fatalf("Words = %#v", words)
	}
	if got := Collapse("  alpha\n\tbeta   gamma  "); got != "alpha beta gamma" {
		t.Fatalf("Collapse = %q", got)
	}
	if got := FirstWords(" alpha\n\tbeta ", 3); got != "alpha\n\tbeta" {
		t.Fatalf("FirstWords short = %q", got)
	}
	if got := FirstWords("  alpha\n beta  gamma delta ", 3); got != "alpha beta gamma" {
		t.Fatalf("FirstWords limited = %q", got)
	}
	if got := Compact(" alpha\n\tbeta  gamma ", 11, "empty"); got != "alpha beta" {
		t.Fatalf("Compact limited = %q", got)
	}
	if got := Compact(" \n\t ", 11, "empty"); got != "empty" {
		t.Fatalf("Compact empty = %q", got)
	}
}

func TestCompactDoesNotSplitUTF8Runes(t *testing.T) {
	got := Compact("alpha 🙂 beta", 8, "empty")
	if !utf8.ValidString(got) {
		t.Fatalf("Compact returned invalid UTF-8 bytes: % x", []byte(got))
	}
	if got != "alpha" {
		t.Fatalf("Compact = %q, want safely truncated text before split emoji", got)
	}
}

func TestTokenBudgetContracts(t *testing.T) {
	if got := WordCount(" alpha\n\tbeta  gamma "); got != 3 {
		t.Fatalf("WordCount = %d, want 3", got)
	}
	if got := ApproxTokens(" \n\t "); got != 1 {
		t.Fatalf("ApproxTokens blank = %d, want 1", got)
	}
	if !FitsTokenBudget(0, 10, 5, true) {
		t.Fatal("expected first item to fit when first-over-budget is allowed")
	}
	if FitsTokenBudget(0, 1, 0, true) {
		t.Fatal("expected disabled budget not to fit")
	}
}

package textutil

import "testing"

func TestCollapseWhitespaceTrimsAndCollapsesUnicodeWhitespace(t *testing.T) {
	got := CollapseWhitespace("  alpha\n\tbeta   gamma  ")
	if got != "alpha beta gamma" {
		t.Fatalf("CollapseWhitespace = %q", got)
	}
}

func TestCollapseWhitespaceEmptyWhenOnlyWhitespace(t *testing.T) {
	got := CollapseWhitespace(" \n\t ")
	if got != "" {
		t.Fatalf("CollapseWhitespace whitespace = %q, want empty", got)
	}
}

func TestFirstWordsLimitsByWhitespaceTokens(t *testing.T) {
	got := FirstWords("  alpha\n beta  gamma delta ", 3)
	if got != "alpha beta gamma" {
		t.Fatalf("FirstWords = %q", got)
	}
}

func TestFirstWordsPreservesTrimmedShortContent(t *testing.T) {
	got := FirstWords(" alpha\n\tbeta ", 3)
	if got != "alpha\n\tbeta" {
		t.Fatalf("FirstWords short = %q", got)
	}
}

func TestCompactWhitespaceCollapsesLimitsAndHandlesEmpty(t *testing.T) {
	if got := CompactWhitespace(" alpha\n\tbeta  gamma ", 11, "empty"); got != "alpha beta" {
		t.Fatalf("CompactWhitespace limited = %q", got)
	}
	if got := CompactWhitespace(" \n\t ", 11, "empty"); got != "empty" {
		t.Fatalf("CompactWhitespace empty = %q", got)
	}
}

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

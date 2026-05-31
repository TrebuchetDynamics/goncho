package textutil

import "testing"

func TestFirstNonBlankTrimsAndPreservesFirstNonBlank(t *testing.T) {
	if got := FirstNonBlank(" ", " alpha ", "beta"); got != "alpha" {
		t.Fatalf("FirstNonBlank() = %q, want alpha", got)
	}
	if got := FirstNonBlank(" ", "\n"); got != "" {
		t.Fatalf("FirstNonBlank() = %q, want empty", got)
	}
}

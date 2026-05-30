package stringutil_test

import (
	"testing"

	"github.com/TrebuchetDynamics/goncho/internal/stringutil"
)

func TestFirstNonEmpty(t *testing.T) {
	t.Run("returns first non-empty", func(t *testing.T) {
		got := stringutil.FirstNonEmpty("", "b", "c")
		if got != "b" {
			t.Fatalf("got %q, want %q", got, "b")
		}
	})
	t.Run("skips whitespace only", func(t *testing.T) {
		got := stringutil.FirstNonEmpty("  ", "\t", "x")
		if got != "x" {
			t.Fatalf("got %q, want %q", got, "x")
		}
	})
	t.Run("returns empty when all blank", func(t *testing.T) {
		got := stringutil.FirstNonEmpty()
		if got != "" {
			t.Fatalf("got %q, want %q", got, "")
		}
	})
	t.Run("returns first non-blank preserving exact value", func(t *testing.T) {
		got := stringutil.FirstNonEmpty("", "  hello  ", "world")
		if got != "hello" {
			t.Fatalf("got %q, want %q", got, "hello")
		}
	})
}

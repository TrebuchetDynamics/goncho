package textutil

import "testing"

func TestEqualFoldTrimmed(t *testing.T) {
	if !EqualFoldTrimmed(" Memory ", "memory") {
		t.Fatalf("EqualFoldTrimmed should ignore surrounding whitespace and case")
	}
	if EqualFoldTrimmed("memory", "message") {
		t.Fatalf("EqualFoldTrimmed should reject different text")
	}
}

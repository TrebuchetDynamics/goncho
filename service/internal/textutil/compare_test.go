package textutil

import "testing"

func TestEqualTrimmed(t *testing.T) {
	if !EqualTrimmed(" * ", "*") {
		t.Fatalf("EqualTrimmed should ignore surrounding whitespace")
	}
	if EqualTrimmed("Memory", "memory") {
		t.Fatalf("EqualTrimmed should remain case-sensitive")
	}
}

func TestEqualFoldTrimmed(t *testing.T) {
	if !EqualFoldTrimmed(" Memory ", "memory") {
		t.Fatalf("EqualFoldTrimmed should ignore surrounding whitespace and case")
	}
	if EqualFoldTrimmed("memory", "message") {
		t.Fatalf("EqualFoldTrimmed should reject different text")
	}
}

func TestContainsTrimmed(t *testing.T) {
	if !ContainsTrimmed([]string{"alpha", " * "}, "*") {
		t.Fatalf("ContainsTrimmed should match trimmed values")
	}
	if ContainsTrimmed([]string{"Memory"}, "memory") {
		t.Fatalf("ContainsTrimmed should remain case-sensitive")
	}
}

func TestContainsEqualFoldTrimmed(t *testing.T) {
	if !ContainsEqualFoldTrimmed([]string{"alpha", " Memory "}, "memory") {
		t.Fatalf("ContainsEqualFoldTrimmed should ignore surrounding whitespace and case")
	}
	if ContainsEqualFoldTrimmed([]string{"memory"}, "message") {
		t.Fatalf("ContainsEqualFoldTrimmed should reject different text")
	}
}

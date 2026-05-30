package idutil

import "testing"

func TestDecimalAndPrefixed(t *testing.T) {
	if got := Decimal(42); got != "42" {
		t.Fatalf("Decimal(42) = %q", got)
	}
	if got := Prefixed("message:", 42); got != "message:42" {
		t.Fatalf("Prefixed = %q", got)
	}
}

func TestParseDecimalTrimsWhitespace(t *testing.T) {
	got, err := ParseDecimal(" 42 \n")
	if err != nil || got != 42 {
		t.Fatalf("ParseDecimal = %d, %v", got, err)
	}
}

func TestParsePrefixedRequiresPrefix(t *testing.T) {
	got, err := ParsePrefixed("conclusion:42", "conclusion:")
	if err != nil || got != 42 {
		t.Fatalf("ParsePrefixed = %d, %v", got, err)
	}
	if _, err := ParsePrefixed("image:42", "conclusion:"); err == nil {
		t.Fatalf("ParsePrefixed accepted wrong prefix")
	}
}

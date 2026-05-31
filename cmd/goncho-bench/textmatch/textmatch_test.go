package textmatch

import "testing"

func TestContainsAnyReportsMatchingSubstring(t *testing.T) {
	if !ContainsAny("latest memory update", []string{"current", "latest"}) {
		t.Fatal("expected matching substring")
	}
}

func TestContainsAnyReportsNoMatch(t *testing.T) {
	if ContainsAny("lexical grounding", []string{"temporal", "numeric"}) {
		t.Fatal("expected no matching substring")
	}
}

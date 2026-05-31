package textutil

import "testing"

func TestMatchesOptionalTrimmed(t *testing.T) {
	if !MatchesOptionalTrimmed("workspace-a", " workspace-a ") {
		t.Fatalf("trimmed matching filter should match value")
	}
	if !MatchesOptionalTrimmed("workspace-a", "   ") {
		t.Fatalf("blank optional filter should match any value")
	}
	if MatchesOptionalTrimmed("workspace-a", "workspace-b") {
		t.Fatalf("different non-empty filter should not match value")
	}
}

func TestMatchesOptionalTrimmedOrEmpty(t *testing.T) {
	if !MatchesOptionalTrimmedOrEmpty("session-a", " session-a ") {
		t.Fatalf("trimmed matching filter should match value")
	}
	if !MatchesOptionalTrimmedOrEmpty("", "session-a") {
		t.Fatalf("empty value should match optional legacy-unscoped filter")
	}
	if MatchesOptionalTrimmedOrEmpty("session-a", "session-b") {
		t.Fatalf("different non-empty filter should not match non-empty value")
	}
}

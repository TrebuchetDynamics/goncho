package shared

import "testing"

func TestNormalizeAbilityTrimsAndUppercases(t *testing.T) {
	if got := NormalizeAbility(" rw "); got != "RW" {
		t.Fatalf("NormalizeAbility() = %q, want RW", got)
	}
}

func TestNormalizeRecordTypeTrimsAndLowercases(t *testing.T) {
	if got := NormalizeRecordType(" Question "); got != "question" {
		t.Fatalf("NormalizeRecordType() = %q, want question", got)
	}
}

func TestNormalizeEvidenceKindTrimsAndLowercases(t *testing.T) {
	if got := NormalizeEvidenceKind(" Graph "); got != "graph" {
		t.Fatalf("NormalizeEvidenceKind() = %q, want graph", got)
	}
}

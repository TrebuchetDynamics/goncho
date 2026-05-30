package textutil

import "testing"

func TestCutBeforeAnySubstringFold(t *testing.T) {
	t.Parallel()

	got, ok := CutBeforeAnySubstringFold("alpha BUT beta because gamma", " because ", " but ")
	if !ok || got != "alpha" {
		t.Fatalf("CutBeforeAnySubstringFold earliest = %q, %v", got, ok)
	}
	got, ok = CutBeforeAnySubstringFold("alpha", "")
	if ok || got != "alpha" {
		t.Fatalf("CutBeforeAnySubstringFold no match = %q, %v", got, ok)
	}
}

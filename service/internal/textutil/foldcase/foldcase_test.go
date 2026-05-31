package foldcase

import "testing"

func TestFoldedMarkerHelpersReuseLowerHaystack(t *testing.T) {
	lower := Lower("Deployment Owner")

	if !ContainsFolded(lower, "OWNER") {
		t.Fatal("expected folded contains match")
	}
	if !HasPrefixFolded(lower, "deployment") {
		t.Fatal("expected folded prefix match")
	}
	if got := IndexFolded(lower, "Owner"); got != len("deployment ") {
		t.Fatalf("IndexFolded = %d, want %d", got, len("deployment "))
	}
}

func TestEitherSubstring(t *testing.T) {
	if !EitherSubstring("Deployment Owner", "owner") {
		t.Fatal("expected case-folded either-direction substring match")
	}
	if EitherSubstring("billing", "auth") {
		t.Fatal("unexpected case-folded either-direction substring match")
	}
}

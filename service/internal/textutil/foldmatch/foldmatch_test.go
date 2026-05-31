package foldmatch

import "testing"

func TestEitherSubstring(t *testing.T) {
	if !EitherSubstring("Deployment Owner", "owner") {
		t.Fatal("expected case-folded either-direction substring match")
	}
	if EitherSubstring("billing", "auth") {
		t.Fatal("unexpected case-folded either-direction substring match")
	}
}

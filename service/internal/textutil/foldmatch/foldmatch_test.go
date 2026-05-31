package foldmatch

import "testing"

func TestPrefixContracts(t *testing.T) {
	if !HasAnyPrefix("Next: ship it", "todo:", "next:") {
		t.Fatal("expected folded prefix match")
	}
	if HasAnyPrefix("ship it", "", "todo:") {
		t.Fatal("expected empty prefix to be ignored by predicate")
	}

	tail, ok := CutAnyPrefix("Where Is Vault", []string{"where is ", "where are "})
	if tail != "Vault" || !ok {
		t.Fatalf("CutAnyPrefix = (%q, %v), want (%q, %v)", tail, ok, "Vault", true)
	}
	tail, ok = CutAnyPrefix("unchanged", []string{""})
	if tail != "unchanged" || !ok {
		t.Fatalf("CutAnyPrefix empty-prefix contract = (%q, %v), want (%q, %v)", tail, ok, "unchanged", true)
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

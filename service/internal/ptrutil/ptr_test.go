package ptrutil

import "testing"

func TestBoolReturnsIndependentPointer(t *testing.T) {
	got := Bool(true)
	if got == nil || !*got {
		t.Fatalf("Bool(true) = %v, want pointer to true", got)
	}
	other := Bool(false)
	if other == nil || *other {
		t.Fatalf("Bool(false) = %v, want pointer to false", other)
	}
	*got = false
	if *other {
		t.Fatalf("Bool pointers alias unexpectedly")
	}
}

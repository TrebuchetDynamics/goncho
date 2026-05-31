package substrmatch

import "testing"

func TestAny(t *testing.T) {
	if !Any("remember the latest plan", []string{"current", "latest"}) {
		t.Fatal("expected one marker to match")
	}
	if Any("remember the plan", []string{"current", "latest"}) {
		t.Fatal("unexpected marker match")
	}
	if Any("remember the latest plan", nil) {
		t.Fatal("nil marker list should not match")
	}
}

func TestEither(t *testing.T) {
	if !Either("vault auth service", "auth") || !Either("auth", "vault auth service") {
		t.Fatal("expected either-direction substring match")
	}
	if Either("Auth", "auth") {
		t.Fatal("Either should remain case-sensitive")
	}
	if Either("billing", "auth") {
		t.Fatal("unexpected either-direction substring match")
	}
}

package substrmatch

import "testing"

func TestAnyMatchUsesMatcherContract(t *testing.T) {
	calls := 0
	matchSuffix := func(value, marker string) bool {
		calls++
		return len(value) >= len(marker) && value[len(value)-len(marker):] == marker
	}

	if !AnyMatch("fact summary", []string{"plan", "summary", "unused"}, matchSuffix) {
		t.Fatal("expected matcher-selected marker to match")
	}
	if calls != 2 {
		t.Fatalf("AnyMatch calls = %d, want 2 to stop at first match", calls)
	}
	if AnyMatch("fact summary", []string{"plan"}, nil) {
		t.Fatal("nil matcher should not match")
	}
}

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

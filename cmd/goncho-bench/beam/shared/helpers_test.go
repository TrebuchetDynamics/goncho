package shared

import "testing"

func TestNormalizeAbilityTrimsAndUppercases(t *testing.T) {
	if got := NormalizeAbility(" rw "); got != "RW" {
		t.Fatalf("NormalizeAbility() = %q, want RW", got)
	}
}

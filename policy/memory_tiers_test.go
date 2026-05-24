package policy

import "testing"

func TestMemoryPolicyPublicFacadeNormalizesAndOrdersTiers(t *testing.T) {
	if got := NormalizeTier(" Project "); got != TierProject {
		t.Fatalf("NormalizeTier = %q, want %q", got, TierProject)
	}
	if !ValidTier("decision") {
		t.Fatal("ValidTier(decision) = false, want true")
	}
	readable := TiersReadableBy(TierTask)
	want := []MemoryTier{TierGlobal, TierProject, TierTask}
	if len(readable) != len(want) {
		t.Fatalf("TiersReadableBy(TierTask) = %v, want %v", readable, want)
	}
	for i := range want {
		if readable[i] != want[i] {
			t.Fatalf("TiersReadableBy(TierTask) = %v, want %v", readable, want)
		}
	}
	if got := DefaultTierForSource("reviewed_proposal"); got != TierProject {
		t.Fatalf("DefaultTierForSource(reviewed_proposal) = %q, want %q", got, TierProject)
	}
}

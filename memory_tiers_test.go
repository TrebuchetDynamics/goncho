package goncho

import "testing"

func TestValidTier(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"global", true}, {"project", true}, {"task", true},
		{"workspace", true}, {"decision", true},
		{"GLOBAL", true}, {" Project ", true},
		{"", false}, {"invalid", false}, {"admin", false},
	}
	for _, tc := range tests {
		got := ValidTier(tc.input)
		if got != tc.want {
			t.Errorf("ValidTier(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestNormalizeTier(t *testing.T) {
	if got := NormalizeTier(""); got != TierGlobal {
		t.Errorf("empty -> %v, want %v", got, TierGlobal)
	}
	if got := NormalizeTier("project"); got != TierProject {
		t.Errorf("project -> %v, want %v", got, TierProject)
	}
	if got := NormalizeTier("UNKNOWN"); got != TierGlobal {
		t.Errorf("unknown -> %v, want %v", got, TierGlobal)
	}
}

func TestTiersReadableBy(t *testing.T) {
	tiers := TiersReadableBy(TierTask)
	if len(tiers) != 3 {
		t.Fatalf("Task agent should read 3 tiers, got %d", len(tiers))
	}
	expected := []MemoryTier{TierGlobal, TierProject, TierTask}
	for i, want := range expected {
		if tiers[i] != want {
			t.Errorf("tiers[%d] = %v, want %v", i, tiers[i], want)
		}
	}
}

func TestTiersWritableBy(t *testing.T) {
	childTiers := TiersWritableBy(false)
	if len(childTiers) != 1 || childTiers[0] != TierWorkspace {
		t.Errorf("child writable tiers = %v, want [workspace]", childTiers)
	}
	parentTiers := TiersWritableBy(true)
	if len(parentTiers) != 5 {
		t.Errorf("parent writable tier count = %d, want 5", len(parentTiers))
	}
}

func TestDefaultTierForSource(t *testing.T) {
	if got := DefaultTierForSource("manual"); got != TierProject {
		t.Errorf("manual -> %v, want project", got)
	}
	if got := DefaultTierForSource("runtime"); got != TierTask {
		t.Errorf("runtime -> %v, want task", got)
	}
	if got := DefaultTierForSource("reviewed_proposal"); got != TierProject {
		t.Errorf("reviewed_proposal -> %v, want project", got)
	}
}

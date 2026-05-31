package retrieval

import "testing"

func TestStableIDsForContentsDeduplicatesByContentOrderAndCapsLimit(t *testing.T) {
	contentIDs := map[string][]string{
		ContentKey("c1", "duplicate"): {"m1", "m2"},
		ContentKey("c1", "unique"):    {"m2", "m3"},
		ContentKey("c2", "duplicate"): {"other"},
	}

	got := StableIDsForContents("c1", []string{"duplicate", "unique"}, contentIDs, 2)
	want := []string{"m1", "m2"}
	if len(got) != len(want) {
		t.Fatalf("StableIDsForContents len = %d (%v), want %d (%v)", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("StableIDsForContents = %v, want %v", got, want)
		}
	}
}

func TestStableIDsForContentsZeroLimitMeansUnbounded(t *testing.T) {
	contentIDs := map[string][]string{ContentKey("c1", "duplicate"): {"m1", "m2"}}
	got := StableIDsForContents("c1", []string{"duplicate"}, contentIDs, 0)
	if len(got) != 2 || got[0] != "m1" || got[1] != "m2" {
		t.Fatalf("StableIDsForContents with zero limit = %v, want [m1 m2]", got)
	}
}

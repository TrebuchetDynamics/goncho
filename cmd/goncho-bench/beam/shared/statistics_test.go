package shared

import "testing"

func TestScoreTallyCountsAndRoundsAverage(t *testing.T) {
	var tally ScoreTally
	if tally.Count() != 0 || tally.Average() != 0 {
		t.Fatalf("empty ScoreTally count/average = %d/%v, want 0/0", tally.Count(), tally.Average())
	}
	tally.Add(0.333333)
	tally.Add(1)
	if tally.Count() != 2 {
		t.Fatalf("ScoreTally.Count() = %d, want 2", tally.Count())
	}
	if got := tally.Average(); got != 0.6667 {
		t.Fatalf("ScoreTally.Average() = %v, want 0.6667", got)
	}
}

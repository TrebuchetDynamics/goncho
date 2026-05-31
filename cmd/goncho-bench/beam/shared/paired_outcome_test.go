package shared

import "testing"

func TestPairedOutcomeCorrectUsesSharedThreshold(t *testing.T) {
	for _, tc := range []struct {
		name  string
		score float64
		want  bool
	}{
		{name: "below threshold", score: 0.49, want: false},
		{name: "at threshold", score: 0.5, want: true},
		{name: "above threshold", score: 1, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := PairedOutcomeCorrect(tc.score); got != tc.want {
				t.Fatalf("PairedOutcomeCorrect(%v) = %v, want %v", tc.score, got, tc.want)
			}
		})
	}
}

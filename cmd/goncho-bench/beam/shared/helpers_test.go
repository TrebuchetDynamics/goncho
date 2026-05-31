package shared

import (
	"testing"
	"time"
)

func TestNormalizeAbilityTrimsAndUppercases(t *testing.T) {
	if got := NormalizeAbility(" rw "); got != "RW" {
		t.Fatalf("NormalizeAbility() = %q, want RW", got)
	}
}

func TestFirstNonEmptyTrimmedReturnsFirstTrimmedValue(t *testing.T) {
	if got := FirstNonEmptyTrimmed("", "  ", " candidate ", "fallback"); got != "candidate" {
		t.Fatalf("FirstNonEmptyTrimmed() = %q, want candidate", got)
	}
	if got := FirstNonEmptyTrimmed("", "  "); got != "" {
		t.Fatalf("FirstNonEmptyTrimmed() with blanks = %q, want empty", got)
	}
}

func TestHasNonEmptyTrimmed(t *testing.T) {
	if HasNonEmptyTrimmed("  ") {
		t.Fatal("HasNonEmptyTrimmed(blank) = true, want false")
	}
	if !HasNonEmptyTrimmed(" value ") {
		t.Fatal("HasNonEmptyTrimmed(value) = false, want true")
	}
}

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

func TestFormatArtifactTimestampUsesUTCRFC3339(t *testing.T) {
	timestamp := time.Date(2026, 5, 30, 10, 15, 0, 0, time.FixedZone("offset", 2*60*60))
	if got := FormatArtifactTimestamp(timestamp); got != "2026-05-30T08:15:00Z" {
		t.Fatalf("FormatArtifactTimestamp() = %q, want UTC RFC3339", got)
	}
}

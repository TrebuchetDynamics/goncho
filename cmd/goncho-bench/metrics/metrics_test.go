package metrics

import "testing"

func TestRoundUsesCanonicalFourDecimalPlaces(t *testing.T) {
	if got := Round(0.12345); got != 0.1235 {
		t.Fatalf("Round(0.12345) = %v, want 0.1235", got)
	}
	if got := Round(1.0 / 3.0); got != 0.3333 {
		t.Fatalf("Round(1/3) = %v, want 0.3333", got)
	}
}

func TestRoundSignedIsSymmetricForDeltas(t *testing.T) {
	if got := RoundSigned(-0.12345); got != -0.1235 {
		t.Fatalf("RoundSigned(-0.12345) = %v, want -0.1235", got)
	}
	if got := RoundSigned(0.12345); got != 0.1235 {
		t.Fatalf("RoundSigned(0.12345) = %v, want 0.1235", got)
	}
}

package providerpolicy

import (
	"testing"
	"time"
)

func TestNormalizeAppliesProviderResilienceDefaults(t *testing.T) {
	got := Normalize(Config{})
	if got.FailureThreshold != DefaultFailureThreshold || got.Cooldown != DefaultCooldown || got.Timeout != DefaultTimeout {
		t.Fatalf("Normalize(Config{}) = %+v, want defaults", got)
	}
}

func TestNormalizePreservesExplicitProviderResilienceValues(t *testing.T) {
	got := Normalize(Config{FailureThreshold: 7, Cooldown: time.Minute, Timeout: 2 * time.Second, MaxPayloadBytes: 42})
	if got.FailureThreshold != 7 || got.Cooldown != time.Minute || got.Timeout != 2*time.Second || got.MaxPayloadBytes != 42 {
		t.Fatalf("Normalize(explicit) = %+v, want explicit values", got)
	}
}

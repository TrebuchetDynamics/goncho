package shared

import (
	"testing"
	"time"
)

func TestFormatArtifactTimestampUsesUTCRFC3339(t *testing.T) {
	timestamp := time.Date(2026, 5, 30, 10, 15, 0, 0, time.FixedZone("offset", 2*60*60))
	if got := FormatArtifactTimestamp(timestamp); got != "2026-05-30T08:15:00Z" {
		t.Fatalf("FormatArtifactTimestamp() = %q, want UTC RFC3339", got)
	}
}

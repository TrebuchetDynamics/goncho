package shared

import "time"

// FormatArtifactTimestamp returns the shared BEAM UTC RFC3339 timestamp format.
func FormatArtifactTimestamp(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

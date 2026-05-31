package shared

import (
	"time"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared/artifactio"
)

// FormatArtifactTimestamp returns the shared BEAM UTC RFC3339 timestamp format.
func FormatArtifactTimestamp(t time.Time) string {
	return artifactio.FormatArtifactTimestamp(t)
}

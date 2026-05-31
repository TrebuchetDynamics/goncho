package shared

import (
	"strings"
	"time"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/checksum"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/metrics"
	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/ranking"
)

func RoundMetric(v float64) float64 {
	return metrics.Round(v)
}

func RoundSignedMetric(v float64) float64 {
	return metrics.RoundSigned(v)
}

func TopN(values []string, n int) []string {
	return ranking.TopN(values, n)
}

func ChecksumBytesSHA256(raw []byte) string {
	return checksum.SHA256Bytes(raw)
}

// FirstNonEmptyTrimmed returns the first argument with non-empty trimmed content.
func FirstNonEmptyTrimmed(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// FormatArtifactTimestamp returns the shared BEAM UTC RFC3339 timestamp format.
func FormatArtifactTimestamp(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

// NormalizeAbility returns the canonical BEAM ability code used in artifacts and match keys.
func NormalizeAbility(ability string) string {
	return strings.ToUpper(strings.TrimSpace(ability))
}

// NormalizeQuestionText returns the canonical question text used for BEAM question-key matching.
func NormalizeQuestionText(question string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(question))), " ")
}

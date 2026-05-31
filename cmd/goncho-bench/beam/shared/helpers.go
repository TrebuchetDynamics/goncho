package shared

import (
	"sort"
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

// SortedStringMapKeys returns map keys in stable lexical order.
func SortedStringMapKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func ChecksumBytesSHA256(raw []byte) string {
	return checksum.SHA256Bytes(raw)
}

// HasNonEmptyTrimmed reports whether value has non-empty content after trimming.
func HasNonEmptyTrimmed(value string) bool {
	return strings.TrimSpace(value) != ""
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

// NormalizeRecordType returns the canonical BEAM JSONL record type.
func NormalizeRecordType(recordType string) string {
	return strings.ToLower(strings.TrimSpace(recordType))
}

// NormalizeEvidenceKind returns the canonical BEAM evidence-kind token.
func NormalizeEvidenceKind(kind string) string {
	return strings.ToLower(strings.TrimSpace(kind))
}

// NormalizeQuestionText returns the canonical question text used for BEAM question-key matching.
func NormalizeQuestionText(question string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(question))), " ")
}

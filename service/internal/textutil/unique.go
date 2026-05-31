package textutil

import (
	"github.com/TrebuchetDynamics/goncho/service/internal/textutil/stringnorm"
	"github.com/TrebuchetDynamics/goncho/service/internal/textutil/trimmed"
)

// Normalizer is the shared contract for callers that canonicalize string values
// before set/unique operations. A nil Normalizer preserves values as-is.
type Normalizer = stringnorm.Normalizer

// NormalizeUnique returns non-empty normalized strings, preserving first-seen
// order unless sortOutput is true.
func NormalizeUnique(values []string, normalize Normalizer, sortOutput bool) []string {
	return stringnorm.Unique(values, normalize, sortOutput)
}

// UniqueTrimmed returns distinct non-empty strings after whitespace trimming.
func UniqueTrimmed(values []string, sortOutput bool) []string {
	return NormalizeUnique(values, trimmed.Space, sortOutput)
}

// UniqueLowerTrimmed returns distinct non-empty strings after trimming and
// lower-casing.
func UniqueLowerTrimmed(values []string, sortOutput bool) []string {
	return NormalizeUnique(values, trimmed.Lower, sortOutput)
}

// Set returns normalized non-empty strings as a set. It preserves nil for empty
// input or when every normalized value is empty.
func Set(values []string, normalize Normalizer) map[string]struct{} {
	return stringnorm.Set(values, normalize)
}

// TrimmedSet returns distinct non-empty strings after whitespace trimming.
func TrimmedSet(values []string) map[string]struct{} {
	return Set(values, trimmed.Space)
}

// LowerTrimmedSet returns distinct non-empty strings after trimming and
// lower-casing.
func LowerTrimmedSet(values []string) map[string]struct{} {
	return Set(values, trimmed.Lower)
}

// SortedSetValues returns the sorted non-empty keys in values after optional
// normalization.
func SortedSetValues(values map[string]struct{}, normalize Normalizer) []string {
	return stringnorm.SortedSetValues(values, normalize)
}

// LowerTrimmed trims surrounding whitespace and applies simple lower-casing.
func LowerTrimmed(value string) string {
	return trimmed.Lower(value)
}

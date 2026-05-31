package policy

const (
	defaultSearchLimit = 10
	maxSearchLimit     = 100
)

// NormalizeLimit applies Goncho's Honcho-compatible search limit policy.
func NormalizeLimit(limit int) int {
	if limit <= 0 {
		return defaultSearchLimit
	}
	if limit > maxSearchLimit {
		return maxSearchLimit
	}
	return limit
}

package searchfilter

const (
	defaultSearchLimit = 10
	maxSearchLimit     = 100
)

func NormalizeLimit(limit int) int {
	if limit <= 0 {
		return defaultSearchLimit
	}
	if limit > maxSearchLimit {
		return maxSearchLimit
	}
	return limit
}

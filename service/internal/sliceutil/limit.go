package sliceutil

// Limit returns values truncated to at most limit items. A non-positive limit
// means no truncation, matching call sites where zero is an unset limit.
func Limit[T any](values []T, limit int) []T {
	if limit > 0 && len(values) > limit {
		return values[:limit]
	}
	return values
}

// LimitClone returns a shallow copy truncated to at most limit items. A
// non-positive limit means no truncation.
func LimitClone[T any](values []T, limit int) []T {
	return Clone(Limit(values, limit))
}

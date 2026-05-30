package sliceutil

// Contains reports whether values includes want.
func Contains[T comparable](values []T, want T) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// ContainsFunc reports whether any value satisfies match.
func ContainsFunc[T any](values []T, match func(T) bool) bool {
	for _, value := range values {
		if match(value) {
			return true
		}
	}
	return false
}

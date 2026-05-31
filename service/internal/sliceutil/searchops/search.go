package searchops

// Contains reports whether values includes want.
func Contains[T comparable](values []T, want T) bool {
	return ContainsFunc(values, func(value T) bool { return value == want })
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

// First returns the first value from values, or the zero value when values is empty.
func First[T any](values []T) T {
	var zero T
	if len(values) == 0 {
		return zero
	}
	return values[0]
}

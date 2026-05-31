package sliceutil

// First returns the first value from values, or the zero value when values is empty.
func First[T any](values []T) T {
	var zero T
	if len(values) == 0 {
		return zero
	}
	return values[0]
}

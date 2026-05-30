package sliceutil

// Clone returns a shallow copy of a slice while preserving nil input as nil.
func Clone[T any](in []T) []T {
	if in == nil {
		return nil
	}
	return append([]T(nil), in...)
}

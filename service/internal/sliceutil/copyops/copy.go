package copyops

// Clone returns a shallow copy of a slice while preserving nil input as nil.
func Clone[T any](in []T) []T {
	if in == nil {
		return nil
	}
	return append([]T(nil), in...)
}

// ReverseClone returns a new slice with values from in in reverse order.
func ReverseClone[T any](in []T) []T {
	out := make([]T, 0, len(in))
	for i := len(in) - 1; i >= 0; i-- {
		out = append(out, in[i])
	}
	return out
}

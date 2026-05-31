package sliceutil

// ReverseClone returns a new slice with values from in in reverse order.
func ReverseClone[T any](in []T) []T {
	out := make([]T, 0, len(in))
	for i := len(in) - 1; i >= 0; i-- {
		out = append(out, in[i])
	}
	return out
}

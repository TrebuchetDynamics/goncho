package sliceutil

// Map returns a slice containing fn applied to each value while preserving nil
// input as nil. A nil fn returns the zero value for each input element.
func Map[T any, U any](values []T, fn func(T) U) []U {
	if values == nil {
		return nil
	}
	out := make([]U, 0, len(values))
	for _, value := range values {
		var mapped U
		if fn != nil {
			mapped = fn(value)
		}
		out = append(out, mapped)
	}
	return out
}

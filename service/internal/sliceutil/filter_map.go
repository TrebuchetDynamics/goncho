package sliceutil

// FilterMap returns mapped values for inputs accepted by fn while preserving nil
// input as nil. A nil fn rejects every value.
func FilterMap[T any, U any](values []T, fn func(T) (U, bool)) []U {
	if values == nil {
		return nil
	}
	out := make([]U, 0, len(values))
	if fn == nil {
		return out
	}
	for _, value := range values {
		mapped, ok := fn(value)
		if ok {
			out = append(out, mapped)
		}
	}
	return out
}

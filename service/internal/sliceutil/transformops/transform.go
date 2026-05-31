package transformops

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

// IndexBy builds an index of the first position for each accepted key in values.
func IndexBy[T any, K comparable](values []T, key func(T) (K, bool)) map[K]int {
	out := make(map[K]int, len(values))
	for i, value := range values {
		k, ok := key(value)
		if !ok {
			continue
		}
		if _, exists := out[k]; exists {
			continue
		}
		out[k] = i
	}
	return out
}

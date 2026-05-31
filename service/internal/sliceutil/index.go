package sliceutil

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

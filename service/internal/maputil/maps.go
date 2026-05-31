package maputil

// Clone returns a shallow copy of a map, preserving nil input as nil.
func Clone[K comparable, V any](in map[K]V) map[K]V {
	if in == nil {
		return nil
	}
	out := make(map[K]V, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

// CloneStringString returns a shallow copy of a string-to-string map.
func CloneStringString(in map[string]string) map[string]string {
	return Clone(in)
}

// CloneStringStringNilIfEmpty returns a shallow copy of a string-to-string map,
// preserving nil/empty input as nil for optional metadata fields.
func CloneStringStringNilIfEmpty(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	return CloneStringString(in)
}

// CloneStringAny returns a shallow copy of a string-to-any map. Empty maps are
// preserved as non-nil to match lifecycle metadata JSON behavior.
func CloneStringAny(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

// CloneStringFloat64 returns a shallow copy of a string-to-float64 map.
func CloneStringFloat64(in map[string]float64) map[string]float64 {
	return Clone(in)
}

// StringStringToAny copies a string-to-string map into a string-to-any map.
func StringStringToAny(in map[string]string) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

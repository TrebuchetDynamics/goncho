package maputil

// CloneStringString returns a shallow copy of a string-to-string map.
func CloneStringString(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
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

package textutil

// CloneStrings returns a shallow copy of a string slice.
func CloneStrings(in []string) []string {
	return append([]string(nil), in...)
}

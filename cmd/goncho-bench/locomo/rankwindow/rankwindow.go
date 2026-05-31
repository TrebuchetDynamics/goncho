package rankwindow

// IDs returns the top-k retrieved IDs without allocating.
func IDs(retrieved []string, k int) []string {
	if k <= 0 {
		return nil
	}
	if k > len(retrieved) {
		k = len(retrieved)
	}
	return retrieved[:k]
}

// IDSet returns a set of non-empty IDs in the top-k retrieved window.
func IDSet(retrieved []string, k int) map[string]struct{} {
	window := IDs(retrieved, k)
	set := make(map[string]struct{}, len(window))
	for _, id := range window {
		if id != "" {
			set[id] = struct{}{}
		}
	}
	return set
}

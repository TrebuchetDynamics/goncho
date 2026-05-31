package shared

import "github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/beam/shared/collection"

// SortedStringMapKeys returns map keys in stable lexical order.
func SortedStringMapKeys[V any](values map[string]V) []string {
	return collection.SortedStringMapKeys(values)
}

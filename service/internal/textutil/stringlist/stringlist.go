package stringlist

import "github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"

// Clone returns a shallow copy of values.
func Clone(values []string) []string {
	return sliceutil.Clone(values)
}

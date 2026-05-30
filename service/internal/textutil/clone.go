package textutil

import "github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"

// CloneStrings returns a shallow copy of a string slice.
func CloneStrings(in []string) []string {
	return sliceutil.Clone(in)
}

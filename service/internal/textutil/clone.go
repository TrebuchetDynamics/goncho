package textutil

import "github.com/TrebuchetDynamics/goncho/service/internal/textutil/stringlist"

// CloneStrings returns a shallow copy of a string slice.
func CloneStrings(in []string) []string {
	return stringlist.Clone(in)
}

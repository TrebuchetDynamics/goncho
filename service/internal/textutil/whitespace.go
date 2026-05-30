package textutil

import (
	"strings"
)

// CollapseWhitespace trims leading/trailing whitespace and converts any run of
// Unicode whitespace to a single ASCII space.
func CollapseWhitespace(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

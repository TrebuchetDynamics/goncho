package foldcase

import "strings"

// Lower applies Goncho's simple lower-case matching policy for substring and
// prefix classifiers. It intentionally does not perform locale-specific
// normalization beyond strings.ToLower.
func Lower(value string) string {
	return strings.ToLower(value)
}

// ContainsFolded reports whether lowerValue contains marker after applying the
// simple lower-case matching policy to marker. lowerValue must already be
// normalized with Lower so callers can reuse one normalized haystack across many
// markers.
func ContainsFolded(lowerValue, marker string) bool {
	return strings.Contains(lowerValue, Lower(marker))
}

// HasPrefixFolded reports whether lowerValue starts with prefix after applying
// the simple lower-case matching policy to prefix. lowerValue must already be
// normalized with Lower.
func HasPrefixFolded(lowerValue, prefix string) bool {
	return strings.HasPrefix(lowerValue, Lower(prefix))
}

// IndexFolded returns the byte index of marker in lowerValue after applying the
// simple lower-case matching policy to marker. lowerValue must already be
// normalized with Lower.
func IndexFolded(lowerValue, marker string) int {
	return strings.Index(lowerValue, Lower(marker))
}

// EitherSubstring reports whether either value contains the other after
// applying Goncho's simple lower-case matching policy to both values.
func EitherSubstring(a, b string) bool {
	lowerA := Lower(a)
	lowerB := Lower(b)
	return strings.Contains(lowerA, lowerB) || strings.Contains(lowerB, lowerA)
}

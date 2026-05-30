package idutil

import (
	"strconv"
	"strings"
)

// Decimal formats a database integer identifier with the canonical base-10
// representation used in public memory IDs, cursors, and stable IDs.
func Decimal(id int64) string {
	return strconv.FormatInt(id, 10)
}

// Prefixed joins a typed stable-ID prefix with a decimal identifier.
func Prefixed(prefix string, id int64) string {
	return prefix + Decimal(id)
}

// ParseDecimal parses a whitespace-trimmed base-10 integer identifier.
func ParseDecimal(value string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(value), 10, 64)
}

// ParsePrefixed parses a decimal identifier after a required prefix.
func ParsePrefixed(value, prefix string) (int64, error) {
	if !strings.HasPrefix(value, prefix) {
		return 0, strconv.ErrSyntax
	}
	return ParseDecimal(strings.TrimPrefix(value, prefix))
}

package utf8limit

import "unicode/utf8"

// TruncateBytes returns value truncated to at most limit bytes without splitting
// a UTF-8 encoded rune. Non-positive limits preserve the historical no-limit
// policy used by preview callers.
func TruncateBytes(value string, limit int) string {
	if limit <= 0 || len([]byte(value)) <= limit {
		return value
	}
	raw := []byte(value)
	if limit > len(raw) {
		limit = len(raw)
	}
	for limit > 0 && !utf8.Valid(raw[:limit]) {
		limit--
	}
	return string(raw[:limit])
}

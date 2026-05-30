package textutil

import "unicode/utf8"

func TruncateUTF8Bytes(value string, limit int) string {
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

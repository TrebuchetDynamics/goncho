package shared

import "bytes"

var jsonNullLiteral = []byte("null")

// TrimJSONRaw returns raw JSON bytes after whitespace trimming.
func TrimJSONRaw(raw []byte) []byte {
	return bytes.TrimSpace(raw)
}

// JSONRawIsEmptyOrNull reports whether a raw JSON value is absent or the JSON null literal.
func JSONRawIsEmptyOrNull(raw []byte) bool {
	trimmed := TrimJSONRaw(raw)
	return len(trimmed) == 0 || bytes.Equal(trimmed, jsonNullLiteral)
}

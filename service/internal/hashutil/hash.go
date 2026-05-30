package hashutil

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// SHA256Hex returns the full lowercase hex SHA-256 digest for raw bytes.
func SHA256Hex(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// SHA256HexString returns the full lowercase hex SHA-256 digest for a string.
func SHA256HexString(value string) string { return SHA256Hex([]byte(value)) }

// SHA256HexPrefix returns the first n bytes of raw's SHA-256 digest as hex.
func SHA256HexPrefix(raw []byte, n int) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:clampedDigestPrefix(n)])
}

// SHA256HexStringPrefix returns the first n bytes of value's SHA-256 digest as hex.
func SHA256HexStringPrefix(value string, n int) string { return SHA256HexPrefix([]byte(value), n) }

// JSONSHA256Hex returns the SHA-256 digest of value's standard JSON encoding.
func JSONSHA256Hex(value any) string {
	raw, _ := json.Marshal(value)
	return SHA256Hex(raw)
}

// JSONSHA256HexPrefix returns the first n bytes of value's JSON SHA-256 digest as hex.
func JSONSHA256HexPrefix(value any, n int) string {
	raw, _ := json.Marshal(value)
	return SHA256HexPrefix(raw, n)
}

func clampedDigestPrefix(n int) int {
	if n < 0 {
		return 0
	}
	if n > sha256.Size {
		return sha256.Size
	}
	return n
}

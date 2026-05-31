package checksum

import (
	"crypto/sha256"
	"encoding/hex"
)

// SHA256Bytes returns the hex-encoded SHA-256 checksum for raw bytes.
func SHA256Bytes(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

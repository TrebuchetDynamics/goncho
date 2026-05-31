package ids

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

var unsafe = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

// Sanitize converts user/session-provided labels into stable plugin IDs.
func Sanitize(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	value = unsafe.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	return value
}

// EnforceLimit keeps a sanitized ID within maxLen, suffixing a digest of original when truncated.
func EnforceLimit(sanitized, original string, maxLen int) string {
	const hashLen = 12
	if maxLen <= hashLen+1 || len(sanitized) <= maxLen {
		return sanitized
	}
	sum := sha256.Sum256([]byte(original))
	digest := hex.EncodeToString(sum[:])[:hashLen]
	prefixLen := maxLen - hashLen - 1
	return strings.TrimRight(sanitized[:prefixLen], "-") + "-" + digest
}

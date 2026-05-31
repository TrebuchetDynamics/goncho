package signing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
)

// ErrSecretMissing reports that no webhook signing secret was supplied.
var ErrSecretMissing = errors.New("goncho: webhook secret is required")

// SignPayload signs a webhook payload using Honcho-compatible HMAC-SHA256 hex.
func SignPayload(payload, secret string) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", ErrSecretMissing
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

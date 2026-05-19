package goncho

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrKeyAuthDisabled  = errors.New("goncho: key creation is disabled when auth is off")
	ErrKeyAdminRequired = errors.New("goncho: scoped key creation requires admin credentials")
	ErrKeyScopeRequired = errors.New("goncho: at least one of workspace_id, peer_id, or session_id is required")
	ErrKeySecretMissing = errors.New("goncho: jwt secret is required")
	ErrKeyInvalid       = errors.New("goncho: invalid scoped key")
	ErrKeyExpired       = errors.New("goncho: scoped key expired")
)

type ScopedKeyParams struct {
	WorkspaceID string
	PeerID      string
	SessionID   string
	ExpiresAt   time.Time
	AuthEnabled bool
	Admin       bool
	Secret      string
	Now         time.Time
}

type ScopedKeyResult struct {
	Key    string          `json:"key"`
	Claims ScopedKeyClaims `json:"claims"`
}

type ScopedKeyClaims struct {
	Timestamp   string `json:"t,omitempty"`
	ExpiresAt   string `json:"exp,omitempty"`
	Admin       *bool  `json:"ad,omitempty"`
	WorkspaceID string `json:"w,omitempty"`
	PeerID      string `json:"p,omitempty"`
	SessionID   string `json:"s,omitempty"`
}

func CreateScopedKey(params ScopedKeyParams) (ScopedKeyResult, error) {
	if !params.AuthEnabled {
		return ScopedKeyResult{}, ErrKeyAuthDisabled
	}
	if !params.Admin {
		return ScopedKeyResult{}, ErrKeyAdminRequired
	}
	claims := ScopedKeyClaims{
		WorkspaceID: strings.TrimSpace(params.WorkspaceID),
		PeerID:      strings.TrimSpace(params.PeerID),
		SessionID:   strings.TrimSpace(params.SessionID),
	}
	if claims.WorkspaceID == "" && claims.PeerID == "" && claims.SessionID == "" {
		return ScopedKeyResult{}, ErrKeyScopeRequired
	}
	now := params.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	claims.Timestamp = now.Format(time.RFC3339)
	if !params.ExpiresAt.IsZero() {
		claims.ExpiresAt = params.ExpiresAt.UTC().Format(time.RFC3339)
	}
	key, err := signScopedKey(claims, params.Secret)
	if err != nil {
		return ScopedKeyResult{}, err
	}
	return ScopedKeyResult{Key: key, Claims: claims}, nil
}

func VerifyScopedKey(token, secret string, now time.Time) (ScopedKeyClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ScopedKeyClaims{}, ErrKeyInvalid
	}
	if strings.TrimSpace(secret) == "" {
		return ScopedKeyClaims{}, ErrKeySecretMissing
	}
	wantSig := scopedKeySignature(parts[0]+"."+parts[1], secret)
	if !hmac.Equal([]byte(parts[2]), []byte(wantSig)) {
		return ScopedKeyClaims{}, ErrKeyInvalid
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ScopedKeyClaims{}, fmt.Errorf("%w: decode payload", ErrKeyInvalid)
	}
	var claims ScopedKeyClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ScopedKeyClaims{}, fmt.Errorf("%w: decode claims", ErrKeyInvalid)
	}
	if claims.ExpiresAt != "" {
		exp, err := time.Parse(time.RFC3339, claims.ExpiresAt)
		if err != nil {
			return ScopedKeyClaims{}, fmt.Errorf("%w: invalid expiration", ErrKeyInvalid)
		}
		if now.IsZero() {
			now = time.Now().UTC()
		}
		if exp.Before(now.UTC()) {
			return ScopedKeyClaims{}, ErrKeyExpired
		}
	}
	return claims, nil
}

func signScopedKey(claims ScopedKeyClaims, secret string) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", ErrKeySecretMissing
	}
	header, err := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	unsigned := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	return unsigned + "." + scopedKeySignature(unsigned, secret), nil
}

func scopedKeySignature(unsigned, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(unsigned))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

package scopedkey

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
	ErrAuthDisabled  = errors.New("goncho: key creation is disabled when auth is off")
	ErrAdminRequired = errors.New("goncho: scoped key creation requires admin credentials")
	ErrScopeRequired = errors.New("goncho: at least one of workspace_id, peer_id, or session_id is required")
	ErrSecretMissing = errors.New("goncho: jwt secret is required")
	ErrInvalid       = errors.New("goncho: invalid scoped key")
	ErrExpired       = errors.New("goncho: scoped key expired")
)

type Params struct {
	WorkspaceID string
	PeerID      string
	SessionID   string
	ExpiresAt   time.Time
	AuthEnabled bool
	Admin       bool
	Secret      string
	Now         time.Time
}

type Result struct {
	Key    string `json:"key"`
	Claims Claims `json:"claims"`
}

type Claims struct {
	Timestamp   string `json:"t,omitempty"`
	ExpiresAt   string `json:"exp,omitempty"`
	Admin       *bool  `json:"ad,omitempty"`
	WorkspaceID string `json:"w,omitempty"`
	PeerID      string `json:"p,omitempty"`
	SessionID   string `json:"s,omitempty"`
}

func Create(params Params) (Result, error) {
	if !params.AuthEnabled {
		return Result{}, ErrAuthDisabled
	}
	if !params.Admin {
		return Result{}, ErrAdminRequired
	}
	claims := Claims{
		WorkspaceID: strings.TrimSpace(params.WorkspaceID),
		PeerID:      strings.TrimSpace(params.PeerID),
		SessionID:   strings.TrimSpace(params.SessionID),
	}
	if claims.WorkspaceID == "" && claims.PeerID == "" && claims.SessionID == "" {
		return Result{}, ErrScopeRequired
	}
	now := params.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	claims.Timestamp = now.Format(time.RFC3339)
	if !params.ExpiresAt.IsZero() {
		claims.ExpiresAt = params.ExpiresAt.UTC().Format(time.RFC3339)
	}
	key, err := sign(claims, params.Secret)
	if err != nil {
		return Result{}, err
	}
	return Result{Key: key, Claims: claims}, nil
}

func Verify(token, secret string, now time.Time) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrInvalid
	}
	if strings.TrimSpace(secret) == "" {
		return Claims{}, ErrSecretMissing
	}
	wantSig := signature(parts[0]+"."+parts[1], secret)
	if !hmac.Equal([]byte(parts[2]), []byte(wantSig)) {
		return Claims{}, ErrInvalid
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, fmt.Errorf("%w: decode payload", ErrInvalid)
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Claims{}, fmt.Errorf("%w: decode claims", ErrInvalid)
	}
	if claims.ExpiresAt != "" {
		exp, err := time.Parse(time.RFC3339, claims.ExpiresAt)
		if err != nil {
			return Claims{}, fmt.Errorf("%w: invalid expiration", ErrInvalid)
		}
		if now.IsZero() {
			now = time.Now().UTC()
		}
		if exp.Before(now.UTC()) {
			return Claims{}, ErrExpired
		}
	}
	return claims, nil
}

func sign(claims Claims, secret string) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", ErrSecretMissing
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
	return unsigned + "." + signature(unsigned, secret), nil
}

func signature(unsigned, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(unsigned))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

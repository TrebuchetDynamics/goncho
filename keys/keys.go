package keys

import (
	"time"

	"github.com/TrebuchetDynamics/goncho/internal/scopedkey"
)

var (
	ErrKeyAuthDisabled  = scopedkey.ErrAuthDisabled
	ErrKeyAdminRequired = scopedkey.ErrAdminRequired
	ErrKeyScopeRequired = scopedkey.ErrScopeRequired
	ErrKeySecretMissing = scopedkey.ErrSecretMissing
	ErrKeyInvalid       = scopedkey.ErrInvalid
	ErrKeyExpired       = scopedkey.ErrExpired
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
	result, err := scopedkey.Create(scopedkey.Params{
		WorkspaceID: params.WorkspaceID,
		PeerID:      params.PeerID,
		SessionID:   params.SessionID,
		ExpiresAt:   params.ExpiresAt,
		AuthEnabled: params.AuthEnabled,
		Admin:       params.Admin,
		Secret:      params.Secret,
		Now:         params.Now,
	})
	if err != nil {
		return ScopedKeyResult{}, err
	}
	return ScopedKeyResult{
		Key:    result.Key,
		Claims: scopedKeyClaimsFromInternal(result.Claims),
	}, nil
}

func VerifyScopedKey(token, secret string, now time.Time) (ScopedKeyClaims, error) {
	claims, err := scopedkey.Verify(token, secret, now)
	if err != nil {
		return ScopedKeyClaims{}, err
	}
	return scopedKeyClaimsFromInternal(claims), nil
}

func scopedKeyClaimsFromInternal(claims scopedkey.Claims) ScopedKeyClaims {
	return ScopedKeyClaims{
		Timestamp:   claims.Timestamp,
		ExpiresAt:   claims.ExpiresAt,
		Admin:       claims.Admin,
		WorkspaceID: claims.WorkspaceID,
		PeerID:      claims.PeerID,
		SessionID:   claims.SessionID,
	}
}

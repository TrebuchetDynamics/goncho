package goncho

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestKeys_CreateScopedKeyRequiresAuthAdminAndScope(t *testing.T) {
	now := time.Date(2026, 4, 28, 14, 0, 0, 0, time.UTC)
	_, err := CreateScopedKey(ScopedKeyParams{
		AuthEnabled: false,
		Admin:       true,
		WorkspaceID: "default",
		Secret:      "secret",
		Now:         now,
	})
	if !errors.Is(err, ErrKeyAuthDisabled) {
		t.Fatalf("disabled auth err = %v, want ErrKeyAuthDisabled", err)
	}

	_, err = CreateScopedKey(ScopedKeyParams{
		AuthEnabled: true,
		Admin:       false,
		WorkspaceID: "default",
		Secret:      "secret",
		Now:         now,
	})
	if !errors.Is(err, ErrKeyAdminRequired) {
		t.Fatalf("non-admin err = %v, want ErrKeyAdminRequired", err)
	}

	_, err = CreateScopedKey(ScopedKeyParams{
		AuthEnabled: true,
		Admin:       true,
		Secret:      "secret",
		Now:         now,
	})
	if !errors.Is(err, ErrKeyScopeRequired) {
		t.Fatalf("missing scope err = %v, want ErrKeyScopeRequired", err)
	}
}

func TestKeys_CreateScopedKeyEmitsHonchoClaimNames(t *testing.T) {
	now := time.Date(2026, 4, 28, 14, 0, 0, 0, time.UTC)
	expires := now.Add(time.Hour)
	got, err := CreateScopedKey(ScopedKeyParams{
		AuthEnabled: true,
		Admin:       true,
		WorkspaceID: "default",
		PeerID:      "alice",
		SessionID:   "sess-1",
		ExpiresAt:   expires,
		Secret:      "secret",
		Now:         now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(got.Key, ".") != 2 {
		t.Fatalf("key = %q, want JWT-shaped token", got.Key)
	}
	claims, err := VerifyScopedKey(got.Key, "secret", now)
	if err != nil {
		t.Fatal(err)
	}
	if claims.WorkspaceID != "default" || claims.PeerID != "alice" || claims.SessionID != "sess-1" {
		t.Fatalf("claims = %+v, want Honcho w/p/s claims", claims)
	}
	if claims.Timestamp != now.Format(time.RFC3339) || claims.ExpiresAt != expires.Format(time.RFC3339) {
		t.Fatalf("timestamps = %q/%q, want RFC3339 t/exp", claims.Timestamp, claims.ExpiresAt)
	}
	if claims.Admin != nil {
		t.Fatalf("Admin claim = %v, want scoped key without ad", *claims.Admin)
	}
}

func TestKeys_CreateAndVerifyScopedKeyRequireSecret(t *testing.T) {
	now := time.Date(2026, 4, 28, 14, 0, 0, 0, time.UTC)
	_, err := CreateScopedKey(ScopedKeyParams{
		AuthEnabled: true,
		Admin:       true,
		WorkspaceID: "default",
		Secret:      "",
		Now:         now,
	})
	if !errors.Is(err, ErrKeySecretMissing) {
		t.Fatalf("missing create secret err = %v, want ErrKeySecretMissing", err)
	}

	got, err := CreateScopedKey(ScopedKeyParams{
		AuthEnabled: true,
		Admin:       true,
		WorkspaceID: "default",
		Secret:      "secret",
		Now:         now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyScopedKey(got.Key, "", now); !errors.Is(err, ErrKeySecretMissing) {
		t.Fatalf("missing verify secret err = %v, want ErrKeySecretMissing", err)
	}
}

func TestKeys_VerifyScopedKeyRejectsWrongSecretAndExpiredToken(t *testing.T) {
	now := time.Date(2026, 4, 28, 14, 0, 0, 0, time.UTC)
	got, err := CreateScopedKey(ScopedKeyParams{
		AuthEnabled: true,
		Admin:       true,
		WorkspaceID: "default",
		ExpiresAt:   now.Add(time.Minute),
		Secret:      "secret",
		Now:         now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyScopedKey(got.Key, "wrong", now); !errors.Is(err, ErrKeyInvalid) {
		t.Fatalf("wrong-secret err = %v, want ErrKeyInvalid", err)
	}
	if _, err := VerifyScopedKey(got.Key, "secret", now.Add(2*time.Minute)); !errors.Is(err, ErrKeyExpired) {
		t.Fatalf("expired err = %v, want ErrKeyExpired", err)
	}
}

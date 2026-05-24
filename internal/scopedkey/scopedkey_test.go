package scopedkey_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/goncho/internal/scopedkey"
)

func TestCreateRequiresAuthAdminAndScope(t *testing.T) {
	now := time.Date(2026, 4, 28, 14, 0, 0, 0, time.UTC)
	_, err := scopedkey.Create(scopedkey.Params{
		AuthEnabled: false,
		Admin:       true,
		WorkspaceID: "default",
		Secret:      "secret",
		Now:         now,
	})
	if !errors.Is(err, scopedkey.ErrAuthDisabled) {
		t.Fatalf("disabled auth err = %v, want ErrAuthDisabled", err)
	}

	_, err = scopedkey.Create(scopedkey.Params{
		AuthEnabled: true,
		Admin:       false,
		WorkspaceID: "default",
		Secret:      "secret",
		Now:         now,
	})
	if !errors.Is(err, scopedkey.ErrAdminRequired) {
		t.Fatalf("non-admin err = %v, want ErrAdminRequired", err)
	}

	_, err = scopedkey.Create(scopedkey.Params{
		AuthEnabled: true,
		Admin:       true,
		Secret:      "secret",
		Now:         now,
	})
	if !errors.Is(err, scopedkey.ErrScopeRequired) {
		t.Fatalf("missing scope err = %v, want ErrScopeRequired", err)
	}
}

func TestCreateEmitsHonchoClaimNames(t *testing.T) {
	now := time.Date(2026, 4, 28, 14, 0, 0, 0, time.UTC)
	expires := now.Add(time.Hour)
	got, err := scopedkey.Create(scopedkey.Params{
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
	claims, err := scopedkey.Verify(got.Key, "secret", now)
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

func TestCreateAndVerifyRequireSecret(t *testing.T) {
	now := time.Date(2026, 4, 28, 14, 0, 0, 0, time.UTC)
	_, err := scopedkey.Create(scopedkey.Params{
		AuthEnabled: true,
		Admin:       true,
		WorkspaceID: "default",
		Secret:      "",
		Now:         now,
	})
	if !errors.Is(err, scopedkey.ErrSecretMissing) {
		t.Fatalf("missing create secret err = %v, want ErrSecretMissing", err)
	}

	got, err := scopedkey.Create(scopedkey.Params{
		AuthEnabled: true,
		Admin:       true,
		WorkspaceID: "default",
		Secret:      "secret",
		Now:         now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := scopedkey.Verify(got.Key, "", now); !errors.Is(err, scopedkey.ErrSecretMissing) {
		t.Fatalf("missing verify secret err = %v, want ErrSecretMissing", err)
	}
}

func TestVerifyRejectsWrongSecretAndExpiredToken(t *testing.T) {
	now := time.Date(2026, 4, 28, 14, 0, 0, 0, time.UTC)
	got, err := scopedkey.Create(scopedkey.Params{
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
	if _, err := scopedkey.Verify(got.Key, "wrong", now); !errors.Is(err, scopedkey.ErrInvalid) {
		t.Fatalf("wrong-secret err = %v, want ErrInvalid", err)
	}
	if _, err := scopedkey.Verify(got.Key, "secret", now.Add(2*time.Minute)); !errors.Is(err, scopedkey.ErrExpired) {
		t.Fatalf("expired err = %v, want ErrExpired", err)
	}
}

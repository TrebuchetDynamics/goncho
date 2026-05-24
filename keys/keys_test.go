package keys

import (
	"strings"
	"testing"
	"time"
)

func TestKeys_PublicFacadeCreatesAndVerifiesScopedKey(t *testing.T) {
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
	if got.Claims.WorkspaceID != "default" || got.Claims.PeerID != "alice" || got.Claims.SessionID != "sess-1" {
		t.Fatalf("created claims = %+v, want public w/p/s claims", got.Claims)
	}

	claims, err := VerifyScopedKey(got.Key, "secret", now)
	if err != nil {
		t.Fatal(err)
	}
	if claims.WorkspaceID != got.Claims.WorkspaceID || claims.PeerID != got.Claims.PeerID || claims.SessionID != got.Claims.SessionID {
		t.Fatalf("verified claims = %+v, want created claims %+v", claims, got.Claims)
	}
	if claims.Timestamp != now.Format(time.RFC3339) || claims.ExpiresAt != expires.Format(time.RFC3339) {
		t.Fatalf("timestamps = %q/%q, want RFC3339 t/exp", claims.Timestamp, claims.ExpiresAt)
	}
	if claims.Admin != nil {
		t.Fatalf("Admin claim = %v, want scoped key without ad", *claims.Admin)
	}
}

package goncho

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGonchoGoalStaleCodeClaimRequiresLiveVerificationE2E(t *testing.T) {
	repo := t.TempDir()
	path := filepath.Join(repo, "src", "security", "auth.ts")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte("export function login() { return true }\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	peer := "peer-code-claims"
	sessionKey := "session-code-claims"
	staleClaim := "Code claim: src/auth.ts contains function login for authentication."
	currentClaim := "Code claim: src/security/auth.ts contains function login for authentication."

	if _, err := svc.Conclude(ctx, ConcludeParams{Peer: peer, Conclusion: staleClaim, SessionKey: sessionKey}); err != nil {
		t.Fatalf("Conclude stale: %v", err)
	}
	if _, err := svc.Conclude(ctx, ConcludeParams{Peer: peer, Conclusion: currentClaim, SessionKey: sessionKey}); err != nil {
		t.Fatalf("Conclude current: %v", err)
	}

	got, err := svc.VerifiedCodeContext(ctx, VerifiedCodeContextParams{
		Peer:       peer,
		SessionKey: sessionKey,
		Query:      "authentication login",
		RepoRoot:   repo,
		MaxTokens:  800,
	})
	if err != nil {
		t.Fatalf("VerifiedCodeContext: %v", err)
	}
	if !strings.Contains(got.Representation, "src/security/auth.ts") {
		t.Fatalf("representation = %q, want verified current path", got.Representation)
	}
	if strings.Contains(got.Representation, "src/auth.ts") {
		t.Fatalf("representation = %q, must not trust stale path", got.Representation)
	}
	if len(got.VerifiedClaims) != 1 || got.VerifiedClaims[0].Path != "src/security/auth.ts" {
		t.Fatalf("verified claims = %+v, want only current auth path", got.VerifiedClaims)
	}
	if len(got.StaleClaims) != 1 || got.StaleClaims[0].Path != "src/auth.ts" {
		t.Fatalf("stale claims = %+v, want stale src/auth.ts", got.StaleClaims)
	}
	if !contextUnavailableHasCapability(got.Unavailable, "stale_code_claim") {
		t.Fatalf("unavailable = %+v, want stale_code_claim evidence", got.Unavailable)
	}
}

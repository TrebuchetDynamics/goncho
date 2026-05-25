package main

import (
	"context"
	"strings"
	"testing"
)

func TestServeRejectsUnauthenticatedNonLoopbackBind(t *testing.T) {
	err := validateServeSecurity(config{Command: "serve", Addr: "0.0.0.0:8765"})
	if err == nil || !strings.Contains(err.Error(), "auth token") || !strings.Contains(err.Error(), "loopback") {
		t.Fatalf("validateServeSecurity unauthenticated public bind err = %v, want loopback/auth-token guidance", err)
	}
}

func TestServeAllowsLoopbackWithoutServerAuth(t *testing.T) {
	if err := validateServeSecurity(config{Command: "serve", Addr: "127.0.0.1:8765"}); err != nil {
		t.Fatalf("loopback validateServeSecurity: %v", err)
	}
	if err := validateServeSecurity(config{Command: "serve", Addr: "localhost:8765"}); err != nil {
		t.Fatalf("localhost validateServeSecurity: %v", err)
	}
}

func TestServeAllowsExplicitAuthTokenForNonLoopbackBind(t *testing.T) {
	if err := validateServeSecurity(config{Command: "serve", Addr: "0.0.0.0:8765", AuthToken: "local-dev-token"}); err != nil {
		t.Fatalf("authenticated public validateServeSecurity: %v", err)
	}
}

func TestRunSecurityPrintsRequirementsWithoutMutation(t *testing.T) {
	var stdout strings.Builder
	if err := run(context.Background(), config{Command: "security", Stdout: &stdout}); err != nil {
		t.Fatalf("run security: %v", err)
	}
	out := stdout.String()
	for _, want := range []string{"requirements_only", "admin", "operator", "reader", "loopback-only", "snapshot manifest"} {
		if !strings.Contains(out, want) {
			t.Fatalf("security report = %q, missing %q", out, want)
		}
	}
}

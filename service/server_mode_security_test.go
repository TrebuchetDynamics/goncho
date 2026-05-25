package goncho

import (
	"slices"
	"strings"
	"testing"
)

func TestServerModeSecurityRequirementsDocumentThreatModelAndRBACVocabulary(t *testing.T) {
	req := ServerModeSecurityRequirements()
	if req.Mode != "server" || req.Status != ServerModeStatusRequirementsOnly || req.EnforcementEnabled {
		t.Fatalf("requirements = %+v, want requirements-only server mode", req)
	}
	for _, want := range []string{"admin", "operator", "reader"} {
		if !slices.Contains(req.Roles, want) {
			t.Fatalf("roles = %v, missing %q", req.Roles, want)
		}
	}
	for _, want := range []string{"auth", "profiles", "workspaces", "audit", "backup", "retention", "admin operations"} {
		if !strings.Contains(strings.ToLower(req.ThreatModelSummary), want) {
			t.Fatalf("threat model summary %q missing %q", req.ThreatModelSummary, want)
		}
	}
	if req.AuthRequirement != "loopback-only unless an explicit server auth token is configured" {
		t.Fatalf("auth requirement = %q, want loopback-only token gate", req.AuthRequirement)
	}
	if !strings.Contains(req.PostgresAdapterPlan, "SQLite remains the reference") || !strings.Contains(req.PostgresAdapterPlan, "conformance") {
		t.Fatalf("postgres plan = %q, want local-first conformance plan", req.PostgresAdapterPlan)
	}
	if !strings.Contains(req.BackupRestoreRequirement, "snapshot manifest") || !strings.Contains(req.BackupRestoreRequirement, "provenance") {
		t.Fatalf("backup requirement = %q, want snapshot/provenance wording", req.BackupRestoreRequirement)
	}
}

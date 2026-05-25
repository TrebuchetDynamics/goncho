package goncho

// ServerModeSecurityStatus describes how much shared-server security is active.
type ServerModeSecurityStatus string

const (
	// ServerModeStatusRequirementsOnly means Goncho has documented server-mode
	// contracts but has not enabled shared/team enforcement yet.
	ServerModeStatusRequirementsOnly ServerModeSecurityStatus = "requirements_only"
)

// ServerModeSecurityRequirement is the public, non-enforcing contract for the
// first server-mode security slice. It intentionally documents the minimum
// controls future shared/team work must satisfy without weakening local mode.
type ServerModeSecurityRequirement struct {
	Mode                     string                   `json:"mode"`
	Status                   ServerModeSecurityStatus `json:"status"`
	EnforcementEnabled       bool                     `json:"enforcement_enabled"`
	Roles                    []string                 `json:"roles"`
	ThreatModelSummary       string                   `json:"threat_model_summary"`
	AuthRequirement          string                   `json:"auth_requirement"`
	PostgresAdapterPlan      string                   `json:"postgres_adapter_plan"`
	BackupRestoreRequirement string                   `json:"backup_restore_requirement"`
}

// ServerModeSecurityRequirements returns the requirements-only threat/RBAC
// contract for future server/team mode. It is safe to surface from CLI/doctor
// flows because it does not grant access or mutate state.
func ServerModeSecurityRequirements() ServerModeSecurityRequirement {
	return ServerModeSecurityRequirement{
		Mode:                     "server",
		Status:                   ServerModeStatusRequirementsOnly,
		EnforcementEnabled:       false,
		Roles:                    []string{"admin", "operator", "reader"},
		ThreatModelSummary:       "Server mode must cover auth, profiles, workspaces, audit, backup, retention, and admin operations before shared/team features are enabled.",
		AuthRequirement:          "loopback-only unless an explicit server auth token is configured",
		PostgresAdapterPlan:      "SQLite remains the reference implementation; any PostgreSQL adapter must pass conformance tests for memory contracts, lifecycle/review state, scoped import/export, and ACL allow/deny decisions.",
		BackupRestoreRequirement: "Backup/export/restore must use snapshot manifest checksums and preserve provenance, lifecycle state, review state, and workspace/profile scope.",
	}
}

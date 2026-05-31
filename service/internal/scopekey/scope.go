package scopekey

import "strings"

// Scope is the normalized workspace/profile/peer key used by team/action records.
type Scope struct {
	WorkspaceID string
	ProfileID   string
	Peer        string
}

// Workspace trims a requested workspace and falls back to defaultWorkspace when
// the request is blank. When wildcardAll is true, "*" maps to the empty
// workspace used by list/audit queries to mean all workspaces.
func Workspace(defaultWorkspace, requested string, wildcardAll bool) string {
	workspaceID := strings.TrimSpace(requested)
	if wildcardAll && workspaceID == "*" {
		return ""
	}
	if workspaceID == "" {
		workspaceID = strings.TrimSpace(defaultWorkspace)
	}
	return workspaceID
}

// Normalize trims scope fields and falls back to defaultWorkspace when workspaceID is blank.
func Normalize(defaultWorkspace, workspaceID, profileID, peer string) Scope {
	return Scope{WorkspaceID: Workspace(defaultWorkspace, workspaceID, false), ProfileID: strings.TrimSpace(profileID), Peer: strings.TrimSpace(peer)}
}

// Complete reports whether the workspace and peer identity are present.
func (s Scope) Complete() bool { return s.WorkspaceID != "" && s.Peer != "" }

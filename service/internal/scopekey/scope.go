package scopekey

import "strings"

// Scope is the normalized workspace/profile/peer key used by team/action records.
type Scope struct {
	WorkspaceID string
	ProfileID   string
	Peer        string
}

// Normalize trims scope fields and falls back to defaultWorkspace when workspaceID is blank.
func Normalize(defaultWorkspace, workspaceID, profileID, peer string) Scope {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		workspaceID = strings.TrimSpace(defaultWorkspace)
	}
	return Scope{WorkspaceID: workspaceID, ProfileID: strings.TrimSpace(profileID), Peer: strings.TrimSpace(peer)}
}

// Complete reports whether the workspace and peer identity are present.
func (s Scope) Complete() bool { return s.WorkspaceID != "" && s.Peer != "" }

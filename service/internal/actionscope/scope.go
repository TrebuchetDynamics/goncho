package actionscope

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/scopekey"
)

// Scope identifies an action within a workspace/profile/peer boundary.
type Scope struct {
	WorkspaceID string
	ProfileID   string
	Peer        string
	ActionID    string
}

// Normalize trims and resolves an action scope against the service default workspace.
func Normalize(defaultWorkspaceID, workspaceID, profileID, peer, actionID string) Scope {
	scope := scopekey.Normalize(defaultWorkspaceID, workspaceID, profileID, peer)
	return Scope{
		WorkspaceID: scope.WorkspaceID,
		ProfileID:   scope.ProfileID,
		Peer:        scope.Peer,
		ActionID:    strings.TrimSpace(actionID),
	}
}

// Complete reports whether the workspace/profile/peer portion is usable.
func (s Scope) Complete() bool {
	return s.WorkspaceID != "" && s.Peer != ""
}

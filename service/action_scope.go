package goncho

import (
	"fmt"
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/scopekey"
)

type scopedAction struct {
	WorkspaceID string
	ProfileID   string
	Peer        string
	ActionID    string
}

func (s *Service) normalizeScopedAction(workspaceID, profileID, peer, actionID, requiredMessage string, requireActionID bool) (scopedAction, error) {
	scope := scopekey.Normalize(s.workspaceID, workspaceID, profileID, peer)
	trimmedActionID := strings.TrimSpace(actionID)
	if !scope.Complete() || (requireActionID && trimmedActionID == "") {
		return scopedAction{}, fmt.Errorf("goncho: %s", requiredMessage)
	}
	return scopedAction{WorkspaceID: scope.WorkspaceID, ProfileID: scope.ProfileID, Peer: scope.Peer, ActionID: trimmedActionID}, nil
}

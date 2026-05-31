package goncho

import (
	"fmt"

	"github.com/TrebuchetDynamics/goncho/service/internal/actionscope"
)

type scopedAction = actionscope.Scope

func (s *Service) normalizeScopedAction(workspaceID, profileID, peer, actionID, requiredMessage string, requireActionID bool) (scopedAction, error) {
	scope := actionscope.Normalize(s.workspaceID, workspaceID, profileID, peer, actionID)
	if !scope.Complete() || (requireActionID && scope.ActionID == "") {
		return scopedAction{}, fmt.Errorf("goncho: %s", requiredMessage)
	}
	return scope, nil
}

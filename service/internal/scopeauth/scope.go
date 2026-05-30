package scopeauth

import (
	"fmt"
	"strings"
)

type ActorScope struct {
	WorkspaceID string
	ProfileID   string
}

func NormalizeActorScope(actorWorkspaceID, actorProfileID, targetWorkspaceID string) ActorScope {
	workspaceID := strings.TrimSpace(actorWorkspaceID)
	if workspaceID == "" {
		workspaceID = targetWorkspaceID
	}
	return ActorScope{WorkspaceID: workspaceID, ProfileID: strings.TrimSpace(actorProfileID)}
}

func SameScope(actor ActorScope, targetWorkspaceID, targetProfileID string) bool {
	return actor.WorkspaceID == targetWorkspaceID && actor.ProfileID == targetProfileID
}

func DeniedReadReason(actor ActorScope, resource, targetWorkspaceID, targetProfileID string) string {
	return fmt.Sprintf("actor scope workspace=%q profile=%q cannot read %s scope workspace=%q profile=%q", actor.WorkspaceID, actor.ProfileID, resource, targetWorkspaceID, targetProfileID)
}

package memoryscope

import (
	"strings"

	"github.com/TrebuchetDynamics/goncho/service/internal/textutil"
)

const (
	Profile   = "profile"
	Workspace = "workspace"
	Shared    = "shared"
	Session   = "session"
	Global    = "global"
)

// Normalize returns a valid memory scope, defaulting to profile when a profile
// is present and workspace otherwise.
func Normalize(scope, profileID string) string {
	scope = textutil.LowerTrimmed(scope)
	switch scope {
	case Profile, Workspace, Shared, Session, Global:
		return scope
	}
	if strings.TrimSpace(profileID) != "" {
		return Profile
	}
	return Workspace
}

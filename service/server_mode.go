package goncho

import (
	"fmt"
	"strings"
)

const (
	ServerModeLocalOnly   = "local-only"
	ServerModeTeamPreview = "team-preview"
	ServerModeTeamEnabled = "team-enabled"
)

func normalizeServerMode(mode string) string {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case "team-preview":
		return ServerModeTeamPreview
	case "team-enabled":
		return ServerModeTeamEnabled
	default:
		return ServerModeLocalOnly
	}
}

func (s *Service) requireDistributedServerMode(capability string) error {
	if s != nil && normalizeServerMode(s.serverMode) == ServerModeTeamEnabled {
		return nil
	}
	return fmt.Errorf("goncho: %s requires server_mode=%s; local embedded mode is not a distributed coordinator", capability, ServerModeTeamEnabled)
}

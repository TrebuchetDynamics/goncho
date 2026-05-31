package workspace

import (
	"errors"
	"strings"
)

// ErrRequired reports that a webhook operation needs a workspace identifier.
var ErrRequired = errors.New("goncho: workspace_id is required")

// Trim normalizes a workspace identifier at package boundaries.
func Trim(workspaceID string) string {
	return strings.TrimSpace(workspaceID)
}

// Resolve returns the explicit workspace when present, otherwise the fallback.
func Resolve(workspaceID, fallback string) string {
	if trimmed := Trim(workspaceID); trimmed != "" {
		return trimmed
	}
	return Trim(fallback)
}

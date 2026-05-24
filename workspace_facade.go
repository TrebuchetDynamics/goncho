package goncho

import workspacepkg "github.com/TrebuchetDynamics/goncho/workspace"

const GlobalWorkspaceID = workspacepkg.GlobalWorkspaceID

// DetectWorkspaceFromPath finds the workspace root by looking for project markers.
// Returns the directory containing the marker and the marker filename.
func DetectWorkspaceFromPath(start string) (workspaceRoot, marker string) {
	return workspacepkg.DetectWorkspaceFromPath(start)
}

// WorkspaceIDForPath returns a stable workspace ID derived from the project root.
func WorkspaceIDForPath(path string) string {
	return workspacepkg.WorkspaceIDForPath(path)
}

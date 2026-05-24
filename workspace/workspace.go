package workspace

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultWorkspaceID = "gormes"
	GlobalWorkspaceID  = "__global__"
)

// DetectWorkspaceFromPath finds the workspace root by looking for project markers.
// Returns the directory containing the marker and the marker filename.
func DetectWorkspaceFromPath(start string) (workspaceRoot, marker string) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", ""
	}

	for {
		for _, m := range []string{".git", "go.mod", "pubspec.yaml", "package.json", "Cargo.toml", "pyproject.toml"} {
			if _, err := os.Stat(filepath.Join(dir, m)); err == nil {
				return dir, m
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", ""
}

// WorkspaceIDForPath returns a stable workspace ID derived from the project root.
func WorkspaceIDForPath(path string) string {
	root, _ := DetectWorkspaceFromPath(path)
	if root == "" {
		return DefaultWorkspaceID
	}
	return "ws-" + safeWorkspaceName(root)
}

func safeWorkspaceName(path string) string {
	name := filepath.Base(path)
	return strings.ToLower(strings.ReplaceAll(name, " ", "-"))
}

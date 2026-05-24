package goncho

import (
	"os"
	"path/filepath"
	"testing"

	workspacepkg "github.com/TrebuchetDynamics/goncho/workspace"
)

func TestWorkspaceDetectionRootFacadePreservesPublicBehavior(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.test/workspace\n"), 0644); err != nil {
		t.Fatalf("WriteFile(go.mod): %v", err)
	}
	nested := filepath.Join(root, "cmd", "agent")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatalf("MkdirAll(nested): %v", err)
	}

	gotRoot, gotMarker := DetectWorkspaceFromPath(nested)
	if gotRoot != root || gotMarker != "go.mod" {
		t.Fatalf("DetectWorkspaceFromPath(%q) = (%q, %q), want (%q, %q)", nested, gotRoot, gotMarker, root, "go.mod")
	}

	if got := WorkspaceIDForPath(nested); got != "ws-"+filepath.Base(root) {
		t.Fatalf("WorkspaceIDForPath(%q) = %q, want %q", nested, got, "ws-"+filepath.Base(root))
	}
	if DefaultWorkspaceID != workspacepkg.DefaultWorkspaceID {
		t.Fatalf("DefaultWorkspaceID = %q, want workspace default %q", DefaultWorkspaceID, workspacepkg.DefaultWorkspaceID)
	}
	if GlobalWorkspaceID != workspacepkg.GlobalWorkspaceID {
		t.Fatalf("GlobalWorkspaceID = %q, want workspace global %q", GlobalWorkspaceID, workspacepkg.GlobalWorkspaceID)
	}
}

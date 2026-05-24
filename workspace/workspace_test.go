package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceDetection_AutoDetectsFromProjectRoot(t *testing.T) {
	tests := []struct {
		name          string
		markers       []string
		nestedDepth   int
		wantMarker    string
		wantWorkspace bool
	}{
		{
			name:          "git repository",
			markers:       []string{".git"},
			wantMarker:    ".git",
			wantWorkspace: true,
		},
		{
			name:          "go module",
			markers:       []string{"go.mod"},
			wantMarker:    "go.mod",
			wantWorkspace: true,
		},
		{
			name:          "flutter project",
			markers:       []string{"pubspec.yaml"},
			wantMarker:    "pubspec.yaml",
			wantWorkspace: true,
		},
		{
			name:          "node project",
			markers:       []string{"package.json"},
			wantMarker:    "package.json",
			wantWorkspace: true,
		},
		{
			name:          "rust project",
			markers:       []string{"Cargo.toml"},
			wantMarker:    "Cargo.toml",
			wantWorkspace: true,
		},
		{
			name:          "python project",
			markers:       []string{"pyproject.toml"},
			wantMarker:    "pyproject.toml",
			wantWorkspace: true,
		},
		{
			name:          "nested directory finds parent marker",
			markers:       []string{".git"},
			nestedDepth:   3,
			wantMarker:    ".git",
			wantWorkspace: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()

			// Create nested directory if needed
			workDir := root
			for i := 0; i < tt.nestedDepth; i++ {
				workDir = filepath.Join(workDir, "level"+string(rune('1'+i)))
				if err := os.MkdirAll(workDir, 0755); err != nil {
					t.Fatal(err)
				}
			}

			// Create marker files in root
			for _, m := range tt.markers {
				path := filepath.Join(root, m)
				if err := os.WriteFile(path, []byte{}, 0644); err != nil {
					t.Fatal(err)
				}
			}

			gotRoot, gotMarker := DetectWorkspaceFromPath(workDir)

			if gotRoot == "" {
				t.Fatalf("DetectWorkspaceFromPath(%q) returned empty root, want workspace root", workDir)
			}
			if gotRoot != root {
				t.Fatalf("DetectWorkspaceFromPath(%q) root = %q, want %q", workDir, gotRoot, root)
			}
			if gotMarker != tt.wantMarker {
				t.Fatalf("DetectWorkspaceFromPath(%q) marker = %q, want %q", workDir, gotMarker, tt.wantMarker)
			}

			// Verify WorkspaceIDForPath produces a stable ID
			wsID := WorkspaceIDForPath(workDir)
			if wsID == DefaultWorkspaceID {
				t.Fatalf("WorkspaceIDForPath(%q) = %q, expected workspace-derived ID", workDir, wsID)
			}
		})
	}
}

func TestWorkspaceDetection_GlobalWorkspaceID(t *testing.T) {
	if GlobalWorkspaceID != "__global__" {
		t.Fatalf("GlobalWorkspaceID = %q, want %q", GlobalWorkspaceID, "__global__")
	}
}

func TestWorkspaceDetection_SafeWorkspaceName(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/home/user/my-project", "my-project"},
		{"/home/user/My Project", "my-project"},
		{"/home/user/UPPERCASE", "uppercase"},
		{"/home/user/mixed_Case", "mixed_case"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := safeWorkspaceName(tt.path)
			if got != tt.want {
				t.Fatalf("safeWorkspaceName(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

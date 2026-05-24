package memorymirror

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublicMirrorDoesNotUseUpstreamProjectNameAsGonchoPackageName(t *testing.T) {
	root := repoRoot(t)
	for _, path := range []string{"README.md", "codemap.md"} {
		raw, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", path, err)
		}
		text := string(raw)
		for _, forbidden := range []string{"goncho/agentmemory", "public `agentmemory` package", "agentmemory.NewToolRegistry", "agentmemory.ArchitectureManifest()"} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("%s still exposes upstream project name as Goncho package API via %q", path, forbidden)
			}
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find go.mod above %s", dir)
		}
		dir = parent
	}
}

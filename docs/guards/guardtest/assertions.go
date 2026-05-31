package guardtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ReadRepoFile reads a repository-relative file for documentation guard tests.
func ReadRepoFile(t *testing.T, path string) string {
	t.Helper()
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(raw)
}

func ContainsAll(t *testing.T, text, label string, markers []string) {
	t.Helper()
	for _, marker := range markers {
		if !strings.Contains(text, marker) {
			t.Fatalf("%s missing %q", label, marker)
		}
	}
}

func ContainsAllFold(t *testing.T, text, label string, markers []string) {
	t.Helper()
	ContainsAll(t, strings.ToLower(text), label, markers)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("locate repository root from %s: go.mod not found", dir)
		}
		dir = parent
	}
}

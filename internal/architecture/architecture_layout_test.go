package architecture

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestArchitectureLayoutScopedKeyImplementationLivesBehindInternalModule(t *testing.T) {
	root := repoRoot(t)

	implPath := filepath.Join(root, "internal", "scopedkey", "scopedkey.go")
	if _, err := os.Stat(implPath); err != nil {
		t.Fatalf("scoped-key implementation must live at %s: %v", implPath, err)
	}

	keysPath := filepath.Join(root, "keys.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), keysPath, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", keysPath, err)
	}
	forbiddenImplementationImports := map[string]struct{}{
		"crypto/hmac":     {},
		"crypto/sha256":   {},
		"encoding/base64": {},
		"encoding/json":   {},
		"errors":          {},
		"fmt":             {},
		"strings":         {},
	}
	for _, imp := range parsed.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			t.Fatalf("Unquote(%s): %v", imp.Path.Value, err)
		}
		if _, forbidden := forbiddenImplementationImports[path]; forbidden {
			t.Fatalf("keys.go imports implementation package %q; keep root keys.go as a public facade and put implementation behind internal/scopedkey", path)
		}
	}
}

func TestArchitectureLayoutWorkspaceDetectionLivesInWorkspacePackage(t *testing.T) {
	root := repoRoot(t)

	for _, rootFile := range []string{"workspace_detection.go", "workspace_detection_test.go"} {
		path := filepath.Join(root, rootFile)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("%s must move out of the root package into the workspace package", path)
		}
	}
	for _, packageFile := range []string{"workspace/workspace.go", "workspace/workspace_test.go"} {
		path := filepath.Join(root, packageFile)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("workspace detection package file missing at %s: %v", path, err)
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

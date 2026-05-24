package architecture

import (
	"bytes"
	"go/ast"
	"go/format"
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

func TestArchitectureLayoutSearchFilterImplementationLivesBehindInternalModule(t *testing.T) {
	root := repoRoot(t)

	implPath := filepath.Join(root, "internal", "searchfilter", "filter.go")
	if _, err := os.Stat(implPath); err != nil {
		t.Fatalf("search-filter implementation must live at %s: %v", implPath, err)
	}

	filterPath := filepath.Join(root, "filter.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), filterPath, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", filterPath, err)
	}
	forbiddenImplementationImports := map[string]struct{}{
		"fmt":     {},
		"slices":  {},
		"strings": {},
	}
	for _, imp := range parsed.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			t.Fatalf("Unquote(%s): %v", imp.Path.Value, err)
		}
		if _, forbidden := forbiddenImplementationImports[path]; forbidden {
			t.Fatalf("filter.go imports implementation package %q; keep root filter.go as a package facade and put implementation behind internal/searchfilter", path)
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

func TestArchitectureLayoutWorkspaceDefaultsHaveSingleOwner(t *testing.T) {
	root := repoRoot(t)

	requireConstExpr(t, filepath.Join(root, "workspace_facade.go"), "DefaultWorkspaceID", "workspacepkg.DefaultWorkspaceID")
	requireConstExpr(t, filepath.Join(root, "workspace_facade.go"), "GlobalWorkspaceID", "workspacepkg.GlobalWorkspaceID")
	forbidConstExpr(t, filepath.Join(root, "topology.go"), "DefaultWorkspaceID")
	requireConstExpr(t, filepath.Join(root, "topology.go"), "EvidenceDefaultWorkspace", `"default_workspace:" + DefaultWorkspaceID`)
	requireConstExpr(t, filepath.Join(root, "integration", "gormes", "adapter.go"), "DefaultWorkspaceID", "goncho.DefaultWorkspaceID")
}

func requireConstExpr(t *testing.T, path, name, want string) {
	t.Helper()
	got, ok := constExpr(t, path, name)
	if !ok {
		t.Fatalf("%s must define const %s as %s", path, name, want)
	}
	if got != want {
		t.Fatalf("%s const %s = %s, want %s", path, name, got, want)
	}
}

func forbidConstExpr(t *testing.T, path, name string) {
	t.Helper()
	if got, ok := constExpr(t, path, name); ok {
		t.Fatalf("%s must not define const %s = %s; keep workspace defaults behind workspace_facade.go", path, name, got)
	}
}

func constExpr(t *testing.T, path, name string) (string, bool) {
	t.Helper()
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("ParseFile(%s): %v", path, err)
	}
	for _, decl := range parsed.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			valueSpec := spec.(*ast.ValueSpec)
			for i, ident := range valueSpec.Names {
				if ident.Name != name {
					continue
				}
				if len(valueSpec.Values) <= i {
					t.Fatalf("%s const %s must have an explicit value", path, name)
				}
				var out bytes.Buffer
				if err := format.Node(&out, fset, valueSpec.Values[i]); err != nil {
					t.Fatalf("format const %s in %s: %v", name, path, err)
				}
				return out.String(), true
			}
		}
	}
	return "", false
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

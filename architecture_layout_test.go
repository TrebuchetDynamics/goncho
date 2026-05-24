package goncho

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestArchitectureLayoutScopedKeyImplementationLivesBehindInternalModule(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

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

package docs_test

import (
	"os"
	"strings"
	"testing"
)

func mustReadGuardFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(raw)
}

func mustContainAll(t *testing.T, text, label string, markers []string) {
	t.Helper()
	for _, marker := range markers {
		if !strings.Contains(text, marker) {
			t.Fatalf("%s missing %q", label, marker)
		}
	}
}

func mustContainAllFold(t *testing.T, text, label string, markers []string) {
	t.Helper()
	mustContainAll(t, strings.ToLower(text), label, markers)
}

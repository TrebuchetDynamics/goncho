package testutil

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func DecodeJSONFile(t testing.TB, path string, out any) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
}

func WriteFile(t testing.TB, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func AssertFileContains(t testing.TB, path, needle string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(raw), needle) {
		t.Fatalf("%s missing %q:\n%s", path, needle, raw)
	}
}

func AssertFileNotContains(t testing.TB, path, unwanted string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if strings.Contains(string(raw), unwanted) {
		t.Fatalf("%s unexpectedly contains %q:\n%s", path, unwanted, raw)
	}
}

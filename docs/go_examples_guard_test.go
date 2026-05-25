package docs_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoExamplesCoverPrimaryP6UseCases(t *testing.T) {
	for _, dir := range []string{"minimal_service", "hook_capture", "recall_trace", "memory_slots", "viewer_server"} {
		path := filepath.Join("..", "examples", "go", dir, "main.go")
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(raw)
		if !strings.Contains(text, "github.com/TrebuchetDynamics/goncho/service") {
			t.Fatalf("%s does not import public service package", path)
		}
	}
}

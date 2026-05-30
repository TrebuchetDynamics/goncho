package docs_test

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestGoExamplesCoverPrimaryP6UseCases(t *testing.T) {
	for _, dir := range []string{"minimal_service", "hook_capture", "recall_trace", "memory_slots", "viewer_server"} {
		path := filepath.Join("..", "..", "examples", "go", dir, "main.go")
		text := mustReadGuardFile(t, path)
		if !strings.Contains(text, "github.com/TrebuchetDynamics/goncho/service") {
			t.Fatalf("%s does not import public service package", path)
		}
	}
}

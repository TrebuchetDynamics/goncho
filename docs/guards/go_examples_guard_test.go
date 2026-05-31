package docs_test

import (
	"path/filepath"
	"testing"
)

func TestGoExamplesCoverPrimaryP6UseCases(t *testing.T) {
	for _, dir := range []string{"minimal_service", "hook_capture", "recall_trace", "memory_slots", "viewer_server"} {
		path := filepath.Join("..", "..", "examples", "go", dir, "main.go")
		mustContainAll(t, mustReadGuardFile(t, path), path, []string{"github.com/TrebuchetDynamics/goncho/service"})
	}
}

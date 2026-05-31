package examples_test

import (
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/goncho/docs/guards/guardtest"
)

func TestGoExamplesCoverPrimaryP6UseCases(t *testing.T) {
	for _, dir := range []string{"minimal_service", "hook_capture", "recall_trace", "memory_slots", "viewer_server"} {
		path := filepath.Join("examples", "go", dir, "main.go")
		guardtest.ContainsAll(t, guardtest.ReadRepoFile(t, path), path, []string{"github.com/TrebuchetDynamics/goncho/service"})
	}
}

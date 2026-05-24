package goncho

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMain(m *testing.M) {
	if cwd, err := os.Getwd(); err == nil && filepath.Base(cwd) == "service" {
		_ = os.Chdir("..")
	}
	os.Exit(m.Run())
}

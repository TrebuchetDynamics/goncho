package shared

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteBytesArtifactCreatesParentDirectories(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "artifact.json")
	if err := WriteBytesArtifact(path, []byte("{}\n"), "make dir", "write artifact"); err != nil {
		t.Fatalf("WriteBytesArtifact() error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != "{}\n" {
		t.Fatalf("WriteBytesArtifact() wrote %q, want %q", got, "{}\n")
	}
}

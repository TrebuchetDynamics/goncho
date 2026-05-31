package shared

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadFileWithChecksumReturnsRawBytesAndSHA256(t *testing.T) {
	path := filepath.Join(t.TempDir(), "artifact.json")
	if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	raw, sha256, err := ReadFileWithChecksum(path, "read artifact")
	if err != nil {
		t.Fatalf("ReadFileWithChecksum() error = %v", err)
	}
	if string(raw) != "{}\n" {
		t.Fatalf("ReadFileWithChecksum() raw = %q, want artifact bytes", raw)
	}
	if sha256 != ChecksumBytesSHA256(raw) {
		t.Fatalf("ReadFileWithChecksum() sha256 = %q, want %q", sha256, ChecksumBytesSHA256(raw))
	}
}

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

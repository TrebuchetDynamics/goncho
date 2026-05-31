package testutil

import (
	"path/filepath"
	"testing"
)

func TestFileAssertionsAndDecodeJSONFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.json")
	WriteFile(t, path, `{"name":"goncho"}`)
	AssertFileContains(t, path, "goncho")
	AssertFileNotContains(t, path, "mnemosyne")

	var row struct {
		Name string `json:"name"`
	}
	DecodeJSONFile(t, path, &row)
	if row.Name != "goncho" {
		t.Fatalf("decoded name = %q, want goncho", row.Name)
	}
}

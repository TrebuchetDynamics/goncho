package shared

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteJSONLRowsEncodesOneRowPerLine(t *testing.T) {
	type row struct {
		Name string `json:"name"`
		N    int    `json:"n"`
	}
	var buf bytes.Buffer
	if err := WriteJSONLRows(&buf, []row{{Name: "alpha", N: 1}, {Name: "beta", N: 2}}, "write rows"); err != nil {
		t.Fatalf("WriteJSONLRows() error = %v", err)
	}
	want := "{\"name\":\"alpha\",\"n\":1}\n{\"name\":\"beta\",\"n\":2}\n"
	if got := buf.String(); got != want {
		t.Fatalf("WriteJSONLRows() = %q, want %q", got, want)
	}
}

func TestWriteJSONLRowsWrapsEncoderErrors(t *testing.T) {
	err := WriteJSONLRows(errWriter{}, []map[string]int{{"n": 1}}, "write rows")
	if err == nil {
		t.Fatal("WriteJSONLRows() error = nil")
	}
	if !strings.Contains(err.Error(), "write rows") || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("WriteJSONLRows() error = %v, want context wrapping boom", err)
	}
}

func TestNewJSONLScannerAcceptsLargePromptRows(t *testing.T) {
	largeLine := strings.Repeat("x", 128*1024)
	scanner := NewJSONLScanner(strings.NewReader(largeLine + "\n"))
	if !scanner.Scan() {
		t.Fatalf("NewJSONLScanner().Scan() = false, err = %v", scanner.Err())
	}
	if got := scanner.Text(); got != largeLine {
		t.Fatalf("NewJSONLScanner().Text() length = %d, want %d", len(got), len(largeLine))
	}
}

func TestReadJSONLFileSkipsBlankLinesAndWrapsDecodeContext(t *testing.T) {
	type row struct {
		Name string `json:"name"`
	}
	path := filepath.Join(t.TempDir(), "rows.jsonl")
	if err := os.WriteFile(path, []byte("{\"name\":\"alpha\"}\n\n{bad}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	_, err := ReadJSONLFile[row](path, "open rows", "scan rows", "decode row")
	if err == nil {
		t.Fatal("ReadJSONLFile() error = nil")
	}
	if !strings.Contains(err.Error(), "decode row line 3") {
		t.Fatalf("ReadJSONLFile() error = %v, want decode context with physical line number", err)
	}

	if err := os.WriteFile(path, []byte("{\"name\":\"alpha\"}\n\n{\"name\":\"beta\"}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	got, err := ReadJSONLFile[row](path, "open rows", "scan rows", "decode row")
	if err != nil {
		t.Fatalf("ReadJSONLFile() error = %v", err)
	}
	if len(got) != 2 || got[0].Name != "alpha" || got[1].Name != "beta" {
		t.Fatalf("ReadJSONLFile() = %#v, want alpha/beta rows", got)
	}
}

func TestWriteAndAppendJSONLFileWithParents(t *testing.T) {
	type row struct {
		Name string `json:"name"`
	}
	path := filepath.Join(t.TempDir(), "nested", "rows.jsonl")
	if err := WriteJSONLFileWithParents(path, "make dir", "create rows", "write rows", []row{{Name: "alpha"}}); err != nil {
		t.Fatalf("WriteJSONLFileWithParents() error = %v", err)
	}
	if err := AppendJSONLFileWithParents(path, "make dir", "open rows", "write rows", []row{{Name: "beta"}}); err != nil {
		t.Fatalf("AppendJSONLFileWithParents() error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	want := "{\"name\":\"alpha\"}\n{\"name\":\"beta\"}\n"
	if string(got) != want {
		t.Fatalf("JSONL file = %q, want %q", got, want)
	}
}

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) {
	return 0, errors.New("boom")
}

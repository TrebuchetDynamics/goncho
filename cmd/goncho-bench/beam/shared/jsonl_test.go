package shared

import (
	"bytes"
	"errors"
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

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) {
	return 0, errors.New("boom")
}

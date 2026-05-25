package docs_test

import (
	"os"
	"strings"
	"testing"
)

func TestPythonHTTPExampleDocumentsStableLocalServerAPI(t *testing.T) {
	raw, err := os.ReadFile("../examples/python/http_recall.py")
	if err != nil {
		t.Fatalf("read python example: %v", err)
	}
	text := string(raw)
	for _, want := range []string{"urllib.request", "127.0.0.1:8765", "/v3/workspaces/", "recall", "goncho-server"} {
		if !strings.Contains(text, want) {
			t.Fatalf("python example missing %q", want)
		}
	}
}

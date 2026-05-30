package docs_test

import (
	"strings"
	"testing"
)

func TestPythonHTTPExampleDocumentsStableLocalServerAPI(t *testing.T) {
	text := mustReadGuardFile(t, "../../examples/python/http_recall.py")
	for _, want := range []string{"urllib.request", "127.0.0.1:8765", "/v3/workspaces/", "recall", "goncho-server"} {
		if !strings.Contains(text, want) {
			t.Fatalf("python example missing %q", want)
		}
	}
}

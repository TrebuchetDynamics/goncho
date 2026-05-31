package docs_test

import "testing"

func TestPythonHTTPExampleDocumentsStableLocalServerAPI(t *testing.T) {
	mustContainAll(t, mustReadGuardFile(t, "../../examples/python/http_recall.py"), "python example", []string{"urllib.request", "127.0.0.1:8765", "/v3/workspaces/", "recall", "goncho-server"})
}

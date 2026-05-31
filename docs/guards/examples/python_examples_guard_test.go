package examples_test

import (
	"testing"

	"github.com/TrebuchetDynamics/goncho/docs/guards/guardtest"
)

func TestPythonHTTPExampleDocumentsStableLocalServerAPI(t *testing.T) {
	guardtest.ContainsAll(t, guardtest.ReadRepoFile(t, "examples/python/http_recall.py"), "python example", []string{"urllib.request", "127.0.0.1:8765", "/v3/workspaces/", "recall", "goncho-server"})
}

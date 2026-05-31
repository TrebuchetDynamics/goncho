package main

import (
	"testing"

	"github.com/TrebuchetDynamics/goncho/cmd/goncho-bench/testutil"
)

func decodeTestJSONFile(t *testing.T, path string, out any) {
	t.Helper()
	testutil.DecodeJSONFile(t, path, out)
}

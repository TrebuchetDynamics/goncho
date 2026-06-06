package release_test

import (
	"testing"

	"github.com/TrebuchetDynamics/goncho/docs/guards/guardtest"
)

func TestComparisonDocsAvoidHypeAndBenchmarkOverclaims(t *testing.T) {
	guardtest.ContainsAllFold(t, guardtest.ReadRepoFile(t, "docs/comparison.md"), "comparison doc", []string{"goncho", "mem0", "agentmemory", "local-first", "evidence", "no star-count", "benchmark claims require"})
}

func TestReleaseChecklistDocumentsSmokeAndPublicVerification(t *testing.T) {
	guardtest.ContainsAll(t, guardtest.ReadRepoFile(t, "docs/release-checklist.md"), "release checklist", []string{"make release-smoke", "make stable-e2e-bench-smoke", "make public-module-smoke", "GitHub release", "pkg.go.dev", "v0.3.2"})
}

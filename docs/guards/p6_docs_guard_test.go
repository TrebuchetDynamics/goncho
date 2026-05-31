package docs_test

import "testing"

func TestComparisonDocsAvoidHypeAndBenchmarkOverclaims(t *testing.T) {
	mustContainAllFold(t, mustReadGuardFile(t, "../comparison.md"), "comparison doc", []string{"goncho", "mem0", "agentmemory", "local-first", "evidence", "no star-count", "benchmark claims require"})
}

func TestReleaseChecklistDocumentsSmokeAndPublicVerification(t *testing.T) {
	mustContainAll(t, mustReadGuardFile(t, "../release-checklist.md"), "release checklist", []string{"make release-smoke", "make stable-e2e-bench-smoke", "make public-module-smoke", "GitHub release", "pkg.go.dev", "v0.3.0"})
}

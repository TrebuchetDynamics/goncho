package docs_test

import (
	"strings"
	"testing"
)

func TestComparisonDocsAvoidHypeAndBenchmarkOverclaims(t *testing.T) {
	doc := strings.ToLower(mustReadGuardFile(t, "../comparison.md"))
	for _, want := range []string{"goncho", "mem0", "agentmemory", "local-first", "evidence", "no star-count", "benchmark claims require"} {
		if !strings.Contains(doc, want) {
			t.Fatalf("comparison doc missing %q", want)
		}
	}
}

func TestReleaseChecklistDocumentsSmokeAndPublicVerification(t *testing.T) {
	doc := mustReadGuardFile(t, "../release-checklist.md")
	for _, want := range []string{"make release-smoke", "make stable-e2e-bench-smoke", "make public-module-smoke", "GitHub release", "pkg.go.dev", "v0.3.0"} {
		if !strings.Contains(doc, want) {
			t.Fatalf("release checklist missing %q", want)
		}
	}
}

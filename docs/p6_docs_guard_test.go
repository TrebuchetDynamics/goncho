package docs_test

import (
	"os"
	"strings"
	"testing"
)

func TestComparisonDocsAvoidHypeAndBenchmarkOverclaims(t *testing.T) {
	raw, err := os.ReadFile("comparison.md")
	if err != nil {
		t.Fatalf("read comparison.md: %v", err)
	}
	doc := strings.ToLower(string(raw))
	for _, want := range []string{"goncho", "mem0", "agentmemory", "local-first", "evidence", "no star-count", "benchmark claims require"} {
		if !strings.Contains(doc, want) {
			t.Fatalf("comparison doc missing %q", want)
		}
	}
}

func TestReleaseChecklistDocumentsSmokeAndPublicVerification(t *testing.T) {
	raw, err := os.ReadFile("release-checklist.md")
	if err != nil {
		t.Fatalf("read release-checklist.md: %v", err)
	}
	doc := string(raw)
	for _, want := range []string{"make release-smoke", "make stable-e2e-bench-smoke", "make public-module-smoke", "GitHub release", "pkg.go.dev", "v0.2.0"} {
		if !strings.Contains(doc, want) {
			t.Fatalf("release checklist missing %q", want)
		}
	}
}

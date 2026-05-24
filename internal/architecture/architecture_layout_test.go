package architecture

import (
	"os"
	"path/filepath"
	"testing"
)

func TestArchitectureLayoutRootHasNoGoFiles(t *testing.T) {
	root := repoRoot(t)
	matches, err := filepath.Glob(filepath.Join(root, "*.go"))
	if err != nil {
		t.Fatalf("Glob root go files: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("repo root must not contain .go files after structured split: %v", matches)
	}
}

func TestArchitectureLayoutStructuredPackagesExist(t *testing.T) {
	root := repoRoot(t)
	wantPackages := []string{
		"service",
		"typedmemory",
		"telemetry",
		"host",
		"keys",
		"dynamicagents",
		"plugins",
		"policy",
		"writequeue",
		"memory",
		"session",
		"workspace",
		"toolmeta",
	}
	for _, pkg := range wantPackages {
		t.Run(pkg, func(t *testing.T) {
			matches, err := filepath.Glob(filepath.Join(root, pkg, "*.go"))
			if err != nil {
				t.Fatalf("Glob(%s): %v", pkg, err)
			}
			if len(matches) == 0 {
				t.Fatalf("structured package %s has no Go files", pkg)
			}
		})
	}
}

func TestArchitectureLayoutInternalImplementationPackagesRemainInternal(t *testing.T) {
	root := repoRoot(t)
	wantInternal := []string{
		"dreamscheduler",
		"dynamicagents",
		"fileimport",
		"hostintegration",
		"importance",
		"localmarkdown",
		"memoryannotations",
		"memorypolicy",
		"observationlog",
		"pluginruntime",
		"queuestatus",
		"reviewlog",
		"scopedkey",
		"searchfilter",
		"skillproposals",
	}
	for _, pkg := range wantInternal {
		t.Run(pkg, func(t *testing.T) {
			matches, err := filepath.Glob(filepath.Join(root, "internal", pkg, "*.go"))
			if err != nil {
				t.Fatalf("Glob(%s): %v", pkg, err)
			}
			if len(matches) == 0 {
				t.Fatalf("internal implementation package %s has no Go files", pkg)
			}
		})
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find go.mod above %s", dir)
		}
		dir = parent
	}
}

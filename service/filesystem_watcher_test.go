package goncho

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
)

func TestFilesystemWatcherPreviewAppliesExplicitIncludeExcludeWithoutWriting(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	root := t.TempDir()
	writeWatcherFixture(t, root, "docs/plan.md", "# Plan\nShip local watcher import.")
	writeWatcherFixture(t, root, "src/main.go", "package main\n")
	writeWatcherFixture(t, root, "node_modules/pkg/index.js", "console.log('skip')\n")
	writeWatcherFixture(t, root, "notes/private.txt", "skip by include\n")

	preview, err := svc.PreviewFilesystemWatcherImport(context.Background(), FilesystemWatcherImportParams{
		RootDir:      root,
		Paths:        []string{filepath.Join(root, "docs/plan.md"), filepath.Join(root, "src/main.go"), filepath.Join(root, "node_modules/pkg/index.js"), filepath.Join(root, "notes/private.txt")},
		IncludeGlobs: []string{"**/*.md", "**/*.go"},
		ExcludeGlobs: []string{"node_modules/**"},
		PeerID:       "fs-watcher",
		SessionKey:   "fs-session",
	})
	if err != nil {
		t.Fatalf("PreviewFilesystemWatcherImport: %v", err)
	}
	if preview.Mutates || preview.RootDir != root || preview.ImportableCount != 2 || preview.SkippedCount != 2 {
		t.Fatalf("preview = %+v, want non-mutating 2 importable/2 skipped", preview)
	}
	gotPaths := watcherCandidatePaths(preview.Candidates)
	if !slices.Equal(gotPaths, []string{"docs/plan.md", "src/main.go"}) {
		t.Fatalf("candidate paths = %v", gotPaths)
	}
	obs, err := svc.ListObservations(context.Background(), ObservationQuery{PeerID: "fs-watcher", Limit: 10})
	if err != nil {
		t.Fatalf("ListObservations: %v", err)
	}
	if obs.Count != 0 {
		t.Fatalf("preview wrote observations: %+v", obs.Observations)
	}
}

func TestFilesystemWatcherImportWritesScopedObservationsWithMetadata(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	root := t.TempDir()
	writeWatcherFixture(t, root, "docs/plan.md", "# Plan\nShip local watcher import.")
	writeWatcherFixture(t, root, "src/main.go", "package main\n")
	writeWatcherFixture(t, root, "dist/bundle.js", "skip bundle\n")

	result, err := svc.ImportFilesystemWatcherChanges(context.Background(), FilesystemWatcherImportParams{
		RootDir:      root,
		Paths:        []string{filepath.Join(root, "docs/plan.md"), filepath.Join(root, "src/main.go"), filepath.Join(root, "dist/bundle.js")},
		IncludeGlobs: []string{"**/*.md", "**/*.go"},
		ExcludeGlobs: []string{"dist/**"},
		PeerID:       "fs-watcher",
		SessionKey:   "fs-session",
	})
	if err != nil {
		t.Fatalf("ImportFilesystemWatcherChanges: %v", err)
	}
	if !result.Mutates || result.ImportedCount != 2 || result.Preview.ImportableCount != 2 {
		t.Fatalf("result = %+v, want two imported observations", result)
	}
	obs, err := svc.ListObservations(context.Background(), ObservationQuery{PeerID: "fs-watcher", SessionKey: "fs-session", Kinds: []ObservationKind{ObservationKindCustom}, Limit: 10})
	if err != nil {
		t.Fatalf("ListObservations: %v", err)
	}
	if obs.Count != 2 {
		t.Fatalf("observations = %+v, want two filesystem watcher observations", obs.Observations)
	}
	paths := []string{obs.Observations[0].Metadata["path"], obs.Observations[1].Metadata["path"]}
	slices.Sort(paths)
	if !slices.Equal(paths, []string{"docs/plan.md", "src/main.go"}) {
		t.Fatalf("observation paths = %v", paths)
	}
	for _, item := range obs.Observations {
		if item.Metadata["connector"] != "filesystem_watcher" || item.Metadata["change_kind"] != "file_change" || item.Metadata["checksum"] == "" {
			t.Fatalf("observation metadata = %+v", item.Metadata)
		}
	}

	replayed, err := svc.ImportFilesystemWatcherChanges(context.Background(), FilesystemWatcherImportParams{
		RootDir:      root,
		Paths:        []string{filepath.Join(root, "docs/plan.md"), filepath.Join(root, "src/main.go")},
		IncludeGlobs: []string{"**/*.md", "**/*.go"},
		PeerID:       "fs-watcher",
		SessionKey:   "fs-session",
	})
	if err != nil {
		t.Fatalf("replay ImportFilesystemWatcherChanges: %v", err)
	}
	if replayed.ReplayedCount != 2 {
		t.Fatalf("replayed = %+v, want deterministic replay count", replayed)
	}
}

func TestFilesystemWatcherRequiresExplicitIncludeRules(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	_, err := svc.PreviewFilesystemWatcherImport(context.Background(), FilesystemWatcherImportParams{RootDir: t.TempDir(), Paths: []string{"README.md"}, PeerID: "fs-watcher", SessionKey: "fs-session"})
	if err == nil {
		t.Fatal("PreviewFilesystemWatcherImport without include globs succeeded, want explicit include rules error")
	}
}

func writeWatcherFixture(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func watcherCandidatePaths(candidates []FilesystemWatcherCandidate) []string {
	out := sliceutil.Map(candidates, func(candidate FilesystemWatcherCandidate) string {
		return candidate.RelativePath
	})
	slices.Sort(out)
	return out
}

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

func TestFilesystemWatcherPreviewDedupesRepeatedPathInputs(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	root := t.TempDir()
	path := filepath.Join(root, "docs/plan.md")
	writeWatcherFixture(t, root, "docs/plan.md", "# Plan\n")

	preview, err := svc.PreviewFilesystemWatcherImport(context.Background(), FilesystemWatcherImportParams{
		RootDir:      root,
		Paths:        []string{path, " " + path + " ", path},
		IncludeGlobs: []string{"**/*.md"},
		PeerID:       "fs-watcher-dupe",
		SessionKey:   "fs-session-dupe",
	})
	if err != nil {
		t.Fatalf("PreviewFilesystemWatcherImport: %v", err)
	}
	if preview.ImportableCount != 1 || preview.SkippedCount != 0 || len(preview.Candidates) != 1 {
		t.Fatalf("preview = %+v, want repeated path input to produce one importable candidate without replay inflation", preview)
	}
	if got := preview.Candidates[0].RelativePath; got != "docs/plan.md" {
		t.Fatalf("candidate relative path = %q, want docs/plan.md", got)
	}
}

func TestFilesystemWatcherImportDoesNotReplayDistinctChangeKinds(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	if err := RunMigrations(svc.db); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	root := t.TempDir()
	path := filepath.Join(root, "docs/plan.md")
	writeWatcherFixture(t, root, "docs/plan.md", "# Plan\n")

	base := FilesystemWatcherImportParams{
		RootDir:      root,
		Paths:        []string{path},
		IncludeGlobs: []string{"**/*.md"},
		PeerID:       "fs-watcher-kind",
		SessionKey:   "fs-session-kind",
	}
	first, err := svc.ImportFilesystemWatcherChanges(context.Background(), base)
	if err != nil {
		t.Fatalf("ImportFilesystemWatcherChanges first: %v", err)
	}
	if first.ImportedCount != 1 || first.ReplayedCount != 0 {
		t.Fatalf("first import = %+v, want fresh file_change observation", first)
	}

	base.ChangeKind = "file_touch"
	second, err := svc.ImportFilesystemWatcherChanges(context.Background(), base)
	if err != nil {
		t.Fatalf("ImportFilesystemWatcherChanges second: %v", err)
	}
	if second.ImportedCount != 1 || second.ReplayedCount != 0 {
		t.Fatalf("second import = %+v, want distinct change_kind observation instead of replay", second)
	}

	obs, err := svc.ListObservations(context.Background(), ObservationQuery{PeerID: "fs-watcher-kind", SessionKey: "fs-session-kind", Kinds: []ObservationKind{ObservationKindCustom}, Limit: 10})
	if err != nil {
		t.Fatalf("ListObservations: %v", err)
	}
	if obs.Count != 2 {
		t.Fatalf("observations = %+v, want both change kinds retained", obs.Observations)
	}
	seen := map[string]bool{}
	for _, item := range obs.Observations {
		seen[item.Metadata["change_kind"]] = true
	}
	if !seen["file_change"] || !seen["file_touch"] {
		t.Fatalf("change kinds = %+v, want file_change and file_touch provenance", seen)
	}
}

func TestFilesystemWatcherPreviewRejectsSymlinkEscapingRoot(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	root := t.TempDir()
	outside := t.TempDir()
	writeWatcherFixture(t, outside, "secret.md", "outside root must not be imported")
	linkPath := filepath.Join(root, "docs", "secret.md")
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret.md"), linkPath); err != nil {
		t.Fatal(err)
	}

	preview, err := svc.PreviewFilesystemWatcherImport(context.Background(), FilesystemWatcherImportParams{
		RootDir:      root,
		Paths:        []string{linkPath},
		IncludeGlobs: []string{"**/*.md"},
		PeerID:       "fs-watcher",
		SessionKey:   "fs-session",
	})
	if err != nil {
		t.Fatalf("PreviewFilesystemWatcherImport: %v", err)
	}
	if preview.ImportableCount != 0 || preview.SkippedCount != 1 || preview.Skipped[0].Reason != "outside_root" {
		t.Fatalf("preview = %+v, want symlink target outside root skipped", preview)
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

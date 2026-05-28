package gormes_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gormes "github.com/TrebuchetDynamics/goncho/integration/gormes"
	goncho "github.com/TrebuchetDynamics/goncho/service"
)

func TestInspectLiveRootSummarizesProfileSessionsAndMemoryWithoutContent(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "sessions", "index.yaml"), sessionIndex("2026-05-28T02:33:22Z", map[string]string{"telegram:redacted-root": "root-session"}, []string{"root-session"}))
	writeFile(t, filepath.Join(root, "profiles", "mineru", "sessions", "index.yaml"), sessionIndex("2026-05-24T02:33:22Z", map[string]string{"telegram:redacted-profile": "profile-session"}, []string{"profile-session", "profile-child"}))
	writeFile(t, filepath.Join(root, "memory", "MEMORY.md"), "Goncho private root clue that must not be returned.\nGormes owner map.\n")
	writeFile(t, filepath.Join(root, "workspace", "memory", "MEMORY.md"), "# Template\nWorkspace memory template.\n")
	writeFile(t, filepath.Join(root, "profiles", "mineru", "GONCHO_MEMORY.md"), "Goncho profile private clue that must not be returned.\n")

	report, err := gormes.InspectLiveRoot(context.Background(), root)
	if err != nil {
		t.Fatalf("InspectLiveRoot: %v", err)
	}

	if report.Root != root || report.RootSessionIndex.SessionCount != 1 || report.RootSessionIndex.LineageCount != 1 {
		t.Fatalf("root report = %+v", report)
	}
	mineru, ok := report.Profile("mineru")
	if !ok {
		t.Fatalf("profiles = %+v, missing mineru", report.Profiles)
	}
	if mineru.SessionIndex.SessionCount != 1 || mineru.SessionIndex.LineageCount != 2 {
		t.Fatalf("mineru session index = %+v", mineru.SessionIndex)
	}
	if len(report.MemoryFiles) != 3 {
		t.Fatalf("memory files = %+v, want root, workspace template, profile", report.MemoryFiles)
	}
	if report.GonchoMentionCount() != 2 {
		t.Fatalf("GonchoMentionCount = %d, want 2", report.GonchoMentionCount())
	}
	projection := goncho.ProjectSessionEvidence(report.SessionEvidenceInput("gormes"))
	projectedMineru, ok := projection.Profile("mineru")
	if !ok || projectedMineru.LineageCount != 2 || projectedMineru.MemoryFileCount != 1 || projection.TotalGonchoMentions != 2 {
		t.Fatalf("projection = %+v projected mineru = %+v", projection, projectedMineru)
	}
	serialized := report.String()
	for _, leaked := range []string{"private root clue", "private profile clue"} {
		if strings.Contains(serialized, leaked) {
			t.Fatalf("report leaked raw memory content %q in %s", leaked, serialized)
		}
	}
}

func sessionIndex(updatedAt string, sessions map[string]string, lineageIDs []string) string {
	var b strings.Builder
	b.WriteString("# Auto-generated session index\nsessions:\n")
	for key, value := range sessions {
		b.WriteString("  ")
		b.WriteString(key)
		b.WriteString(": ")
		b.WriteString(value)
		b.WriteString("\n")
	}
	b.WriteString("lineage:\n")
	for _, id := range lineageIDs {
		b.WriteString("  ")
		b.WriteString(id)
		b.WriteString(":\n    lineage_kind: primary\n    lineage_status: ok\n")
	}
	b.WriteString("updated_at: ")
	b.WriteString(updatedAt)
	b.WriteString("\n")
	return b.String()
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

package goncho

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"
)

func TestGonchoGoalPublicToolsSurviveSQLiteRestartE2E(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	report, err := RunGonchoPublicToolsRestartE2E(ctx, GonchoPublicToolsRestartE2EConfig{
		DBPath:       filepath.Join(dir, "goncho.db"),
		MarkdownPath: filepath.Join(dir, "GONCHO_HANDOFF.md"),
	})
	if err != nil {
		t.Fatalf("RunGonchoPublicToolsRestartE2E: %v", err)
	}

	for _, tool := range []string{"goncho_context", "goncho_search", "goncho_recall", "goncho_remember", "goncho_review", "goncho_handoff"} {
		if !containsPublicRestartValue(report.ToolNames, tool) {
			t.Fatalf("tool names = %#v, missing %s", report.ToolNames, tool)
		}
	}
	if !report.SQLiteRestartVerified || report.NetworkRequired || report.OllamaRequired {
		t.Fatalf("local restart flags = %+v", report)
	}
	if report.SearchCountBeforeRestart != 1 || report.SearchCountAfterRestart != 1 {
		t.Fatalf("search counts = before %d after %d, want 1/1", report.SearchCountBeforeRestart, report.SearchCountAfterRestart)
	}
	if report.RecallSelectedAfterRestart != 1 {
		t.Fatalf("recall selected after restart = %d, want 1", report.RecallSelectedAfterRestart)
	}
	if report.ContextRepresentationAfterRestart == "" {
		t.Fatalf("context representation after restart is empty: %+v", report)
	}
	if !report.ReviewWarningBeforeResolve || report.ReviewWarningAfterResolve {
		t.Fatalf("review warning flags = before %t after %t, want true/false", report.ReviewWarningBeforeResolve, report.ReviewWarningAfterResolve)
	}
	if report.HandoffCountAfterRestart != 1 {
		t.Fatalf("handoff count after restart = %d, want 1", report.HandoffCountAfterRestart)
	}
	if report.CompletionCondition != "go test ./..." {
		t.Fatalf("completion condition = %q", report.CompletionCondition)
	}
}

func containsPublicRestartValue(values []string, want string) bool {
	return sliceutil.Contains(values, want)
}

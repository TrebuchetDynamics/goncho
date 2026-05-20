package gonchohttp

import (
	"context"
	"path/filepath"
	"testing"
)

func TestLocalE2E_HTTPServiceLifecycleSurvivesSQLiteRestart(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	report, err := RunHTTPServiceRestartE2E(ctx, HTTPServiceRestartE2EConfig{
		DBPath: filepath.Join(dir, "goncho-http.db"),
	})
	if err != nil {
		t.Fatalf("RunHTTPServiceRestartE2E: %v", err)
	}
	if !report.SQLiteRestartVerified || report.NetworkRequired || report.ExternalProviderRequired {
		t.Fatalf("restart flags = %+v, want local SQLite-only restart", report)
	}
	if report.MessagesCreated != 2 {
		t.Fatalf("messages created = %d, want 2", report.MessagesCreated)
	}
	if report.SearchCountBeforeRestart != 1 || report.SearchCountAfterRestart != 1 {
		t.Fatalf("search counts before/after = %d/%d, want 1/1", report.SearchCountBeforeRestart, report.SearchCountAfterRestart)
	}
	if !report.ContextHadProfileAfterRestart || !report.ContextHadConclusionAfterRestart || !report.ContextHadRecentMessageAfterRestart {
		t.Fatalf("context flags after restart = %+v", report)
	}
	if report.CompletionCondition != "go test ./..." {
		t.Fatalf("completion condition = %q", report.CompletionCondition)
	}
}

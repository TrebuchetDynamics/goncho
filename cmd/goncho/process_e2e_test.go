package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/TrebuchetDynamics/goncho/memory"
	goncho "github.com/TrebuchetDynamics/goncho/service"
)

func TestProcessE2E_GonchoOperatorCommandsUseRealCLI(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	bin := filepath.Join(t.TempDir(), "goncho-process-e2e")
	if err := exec.CommandContext(ctx, "go", "build", "-o", bin, ".").Run(); err != nil {
		t.Fatalf("go build ./cmd/goncho: %v", err)
	}

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "goncho.db")
	store, err := memory.OpenSqlite(dbPath, 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	if err := goncho.RunMigrations(store.DB()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	if err := store.Close(ctx); err != nil {
		t.Fatalf("Close store: %v", err)
	}

	var schema schemaFingerprintReport
	runGonchoProcessJSON(t, ctx, bin, &schema, "schema-fingerprint", "--json")
	if schema.Service != "goncho" || schema.PublicToolCount != 6 || schema.Fingerprint == "" || schema.Mutates {
		t.Fatalf("schema fingerprint = %+v, want real process non-mutating tool/schema report", schema)
	}
	if !slices.Contains(schema.PublicToolNames, "goncho_recall") {
		t.Fatalf("schema public tools = %v, missing goncho_recall", schema.PublicToolNames)
	}

	var doctor doctorReport
	runGonchoProcessJSON(t, ctx, bin, &doctor, "doctor", "--json", "--db", dbPath, "--config", filepath.Join(dir, "preferences.json"))
	if doctor.Status != "ok" || doctor.Mutates || doctor.DBPath != dbPath {
		t.Fatalf("doctor = %+v, want ok non-mutating migrated DB report", doctor)
	}
	for _, want := range []string{"db_path", "migrations", "preferences", "public_tools"} {
		check, ok := doctor.CheckByName(want)
		if !ok || check.Status != "ok" {
			t.Fatalf("doctor check %q = %+v ok=%v, want ok", want, check, ok)
		}
	}

	watchRoot := filepath.Join(dir, "workspace")
	watchDoc := filepath.Join(watchRoot, "docs", "proof.md")
	if err := os.MkdirAll(filepath.Dir(watchDoc), 0o700); err != nil {
		t.Fatalf("mkdir watcher doc: %v", err)
	}
	if err := os.WriteFile(watchDoc, []byte("# Proof\nProcess E2E plans filesystem watcher import."), 0o600); err != nil {
		t.Fatalf("write watcher doc: %v", err)
	}
	var plan connectPlan
	runGonchoProcessJSON(t, ctx, bin, &plan, "connect", "filesystem-watcher", "--plan", "--watch-root", watchRoot, "--include", "**/*.md", "--exclude", ".git/**")
	if plan.Status != "plan" || plan.Operation != "connect" || plan.Integration != "filesystem-watcher" || plan.Mutates || plan.ConfigAction != "preview_import_changed_files" {
		t.Fatalf("filesystem watcher plan = %+v, want non-mutating real process plan", plan)
	}
	if !slices.Equal(plan.WatchRoots, []string{watchRoot}) || !slices.Contains(plan.IncludeGlobs, "**/*.md") || !strings.Contains(strings.Join(plan.RecommendedNextStep, "\n"), "ImportFilesystemWatcherChanges") {
		t.Fatalf("filesystem watcher plan details = %+v", plan)
	}

	var upgrade upgradeCheckReport
	runGonchoProcessJSON(t, ctx, bin, &upgrade, "upgrade-check", "--json", "--current", "v0.2.0", "--latest", "v0.3.0")
	if upgrade.Status != "update_available" || !upgrade.UpdateAvailable || upgrade.Mutates {
		t.Fatalf("upgrade check = %+v, want non-mutating update_available", upgrade)
	}
}

func runGonchoProcessJSON(t *testing.T, ctx context.Context, bin string, out any, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(ctx, bin, args...)
	raw, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", bin, strings.Join(args, " "), err, raw)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		t.Fatalf("decode %s %s JSON: %v\n%s", bin, strings.Join(args, " "), err, raw)
	}
}

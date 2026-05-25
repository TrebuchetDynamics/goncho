package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/TrebuchetDynamics/goncho/memory"
	goncho "github.com/TrebuchetDynamics/goncho/service"
)

func TestRunSchemaFingerprintReportsStableDriftMetadata(t *testing.T) {
	var stdout bytes.Buffer
	if err := run(context.Background(), config{Command: "schema-fingerprint", SchemaFingerprintJSON: true, Stdout: &stdout}); err != nil {
		t.Fatalf("run schema-fingerprint: %v", err)
	}
	var report schemaFingerprintReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode schema fingerprint: %v\n%s", err, stdout.String())
	}
	if report.Service != "goncho" || report.DBSchemaVersion != goncho.GonchoSQLiteSchemaVersion || report.PublicToolCount != 6 || report.Fingerprint == "" || report.Mutates {
		t.Fatalf("schema fingerprint = %+v, want non-mutating service/schema/tools/fingerprint", report)
	}
	if len(report.HostHookEvents) == 0 {
		t.Fatalf("schema fingerprint missing host hook event names: %+v", report)
	}
}

func TestRunUpgradeCheckReportsAvailableReleaseWithoutMutation(t *testing.T) {
	var stdout bytes.Buffer
	if err := run(context.Background(), config{Command: "upgrade-check", UpgradeJSON: true, CurrentVersion: "v0.2.0", LatestVersion: "v0.2.1", Stdout: &stdout}); err != nil {
		t.Fatalf("run upgrade-check: %v", err)
	}
	var report upgradeCheckReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode upgrade report: %v\n%s", err, stdout.String())
	}
	if report.Status != "update_available" || !report.UpdateAvailable || report.CurrentVersion != "v0.2.0" || report.LatestVersion != "v0.2.1" || report.Mutates {
		t.Fatalf("upgrade report = %+v, want non-mutating update_available", report)
	}
	if len(report.NextSteps) == 0 || !strings.Contains(strings.Join(report.NextSteps, "\n"), "release") {
		t.Fatalf("next steps = %v, want release verification guidance", report.NextSteps)
	}
}

func TestRunDoctorReportsLocalEnvironmentAndMigratedDB(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "goncho.db")
	store, err := memory.OpenSqlite(dbPath, 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	if err := goncho.RunMigrations(store.DB()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	if err := store.Close(context.Background()); err != nil {
		t.Fatalf("close store: %v", err)
	}
	var stdout bytes.Buffer
	if err := run(context.Background(), config{Command: "doctor", DoctorJSON: true, DatabasePath: dbPath, Stdout: &stdout}); err != nil {
		t.Fatalf("run doctor --json: %v", err)
	}
	var report doctorReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode doctor report: %v\n%s", err, stdout.String())
	}
	if report.Status != "ok" || report.DBPath != dbPath || report.Mutates {
		t.Fatalf("doctor report = %+v, want ok non-mutating db path", report)
	}
	for _, name := range []string{"db_path", "migrations", "preferences", "public_tools"} {
		check, ok := report.CheckByName(name)
		if !ok || check.Status != "ok" {
			t.Fatalf("doctor check %q = %+v ok=%v, want ok", name, check, ok)
		}
	}
}

func TestRunDoctorReportsMissingDBWithoutCreatingIt(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "missing", "goncho.db")
	var stdout bytes.Buffer
	if err := run(context.Background(), config{Command: "doctor", DoctorJSON: true, DatabasePath: dbPath, Stdout: &stdout}); err != nil {
		t.Fatalf("run doctor missing db: %v", err)
	}
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Fatalf("doctor created missing db or unexpected stat error: %v", err)
	}
	var report doctorReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode doctor report: %v\n%s", err, stdout.String())
	}
	check, ok := report.CheckByName("db_path")
	if report.Status != "error" || !ok || check.Status != "error" || len(check.Suggestions) == 0 {
		t.Fatalf("missing db report = %+v check=%+v ok=%v, want error with suggestions", report, check, ok)
	}
}

func TestRunVersionJSONReportsModuleSchemaAndToolCount(t *testing.T) {
	var stdout bytes.Buffer
	if err := run(context.Background(), config{Command: "version", VersionJSON: true, Stdout: &stdout}); err != nil {
		t.Fatalf("run version --json: %v", err)
	}
	var report versionReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode version report: %v\n%s", err, stdout.String())
	}
	if report.Service != "goncho" || report.ModuleVersion == "" || report.DBSchemaVersion != goncho.GonchoSQLiteSchemaVersion {
		t.Fatalf("version report = %+v, want service/module/schema", report)
	}
	if report.PublicToolCount != 6 {
		t.Fatalf("public_tool_count = %d, want 6", report.PublicToolCount)
	}
	if report.Mutates {
		t.Fatalf("version report mutates = true")
	}
}

func TestRunPreferencesWritesAndReadsLocalDefaults(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "goncho-preferences.json")
	var writeOut bytes.Buffer

	if err := run(context.Background(), config{
		Command:               "preferences",
		PreferencesConfigPath: configPath,
		PreferenceUpdates: map[string]string{
			"db_path":              filepath.Join(dir, "goncho.db"),
			"workspace_id":         "local-workspace",
			"profile_id":           "operator-profile",
			"redaction_policy":     "strict",
			"connector_permission": "plan_only",
			"bind_addr":            "127.0.0.1:8799",
		},
		Stdout: &writeOut,
	}); err != nil {
		t.Fatalf("write preferences: %v", err)
	}
	var written preferencesReport
	if err := json.Unmarshal(writeOut.Bytes(), &written); err != nil {
		t.Fatalf("decode written preferences: %v\n%s", err, writeOut.String())
	}
	if written.Status != "ok" || !written.Mutates || written.ConfigPath != configPath || written.Preferences.DBPath != filepath.Join(dir, "goncho.db") || written.Preferences.ConnectorPermission != "plan_only" {
		t.Fatalf("written preferences = %+v", written)
	}

	var readOut bytes.Buffer
	if err := run(context.Background(), config{Command: "preferences", PreferencesConfigPath: configPath, Stdout: &readOut}); err != nil {
		t.Fatalf("read preferences: %v", err)
	}
	var read preferencesReport
	if err := json.Unmarshal(readOut.Bytes(), &read); err != nil {
		t.Fatalf("decode read preferences: %v\n%s", err, readOut.String())
	}
	if read.Status != "ok" || read.Mutates || read.Preferences.WorkspaceID != "local-workspace" || read.Preferences.RedactionPolicy != "strict" || read.Preferences.BindAddr != "127.0.0.1:8799" {
		t.Fatalf("read preferences = %+v", read)
	}
}

func TestRunConnectFilesystemWatcherPlanRequiresExplicitIncludeExclude(t *testing.T) {
	root := t.TempDir()
	var stdout bytes.Buffer
	if err := run(context.Background(), config{Command: "connect", Connector: "filesystem-watcher", Plan: true, WatchRoots: []string{root}, IncludeGlobs: []string{"**/*.md", "**/*.go"}, ExcludeGlobs: []string{".git/**", "node_modules/**"}, Stdout: &stdout}); err != nil {
		t.Fatalf("connect filesystem-watcher --plan: %v", err)
	}
	var plan connectPlan
	if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil {
		t.Fatalf("decode filesystem watcher plan: %v\n%s", err, stdout.String())
	}
	if plan.Status != "plan" || plan.Integration != "filesystem-watcher" || plan.Mutates || plan.ConfigAction != "preview_import_changed_files" {
		t.Fatalf("plan = %+v, want non-mutating filesystem watcher preview plan", plan)
	}
	if !slices.Equal(plan.WatchRoots, []string{root}) || !slices.Contains(plan.IncludeGlobs, "**/*.md") || !slices.Contains(plan.ExcludeGlobs, "node_modules/**") {
		t.Fatalf("watch globs = roots:%v include:%v exclude:%v", plan.WatchRoots, plan.IncludeGlobs, plan.ExcludeGlobs)
	}
	if !strings.Contains(strings.Join(plan.RecommendedNextStep, "\n"), "ImportFilesystemWatcherChanges") {
		t.Fatalf("next steps = %v, want service import guidance", plan.RecommendedNextStep)
	}
}

func TestRunConnectFilesystemWatcherRejectsMissingIncludeGlobs(t *testing.T) {
	err := run(context.Background(), config{Command: "connect", Connector: "filesystem-watcher", Plan: true, WatchRoots: []string{t.TempDir()}, Stdout: &bytes.Buffer{}})
	if err == nil {
		t.Fatal("connect filesystem-watcher without include globs succeeded, want explicit include rule error")
	}
}

func TestRunConnectPlanAliasAndRemovePlanAreNonMutating(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".codex", "config.toml")
	var connectOut bytes.Buffer
	if err := run(context.Background(), config{Command: "connect", Connector: "codex", Plan: true, ConfigPath: configPath, ServerAddr: "127.0.0.1:8799", Stdout: &connectOut}); err != nil {
		t.Fatalf("connect --plan: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(configPath)); !os.IsNotExist(err) {
		t.Fatalf("connect --plan created config dir or unexpected stat error: %v", err)
	}
	var connect connectPlan
	if err := json.Unmarshal(connectOut.Bytes(), &connect); err != nil {
		t.Fatalf("decode connect plan: %v\n%s", err, connectOut.String())
	}
	if connect.Status != "plan" || connect.Integration != "codex" || connect.Operation != "connect" || connect.Mutates || connect.ConfigAction != "append_or_replace_mcp_server" {
		t.Fatalf("connect plan = %+v", connect)
	}

	var removeOut bytes.Buffer
	if err := run(context.Background(), config{Command: "remove", Connector: "codex", Plan: true, ConfigPath: configPath, Stdout: &removeOut}); err != nil {
		t.Fatalf("remove --plan: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(configPath)); !os.IsNotExist(err) {
		t.Fatalf("remove --plan created config dir or unexpected stat error: %v", err)
	}
	var remove connectPlan
	if err := json.Unmarshal(removeOut.Bytes(), &remove); err != nil {
		t.Fatalf("decode remove plan: %v\n%s", err, removeOut.String())
	}
	if remove.Status != "plan" || remove.Integration != "codex" || remove.Operation != "remove" || remove.Mutates || remove.ConfigAction != "remove_mcp_server" || !strings.Contains(remove.ConfigPatch, "[mcp_servers.goncho]") {
		t.Fatalf("remove plan = %+v", remove)
	}
}

func TestRunConnectPiDryRunPrintsExtensionPlanWithoutMutating(t *testing.T) {
	dir := t.TempDir()
	settingsPath := filepath.Join(dir, ".pi", "agent", "settings.json")
	extensionPath := filepath.Join(dir, ".pi", "agent", "extensions", "goncho")
	var stdout bytes.Buffer

	err := run(context.Background(), config{
		Command:       "connect",
		Connector:     "pi",
		DryRun:        true,
		ConfigPath:    settingsPath,
		ExtensionPath: extensionPath,
		ServerAddr:    "127.0.0.1:8719",
		Stdout:        &stdout,
	})
	if err != nil {
		t.Fatalf("run connect pi dry-run: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(settingsPath)); !os.IsNotExist(err) {
		t.Fatalf("dry-run created pi settings dir or unexpected stat error: %v", err)
	}
	if _, err := os.Stat(extensionPath); !os.IsNotExist(err) {
		t.Fatalf("dry-run created pi extension dir or unexpected stat error: %v", err)
	}

	var plan connectPlan
	if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil {
		t.Fatalf("decode plan: %v\n%s", err, stdout.String())
	}
	if plan.Status != "dry_run" || plan.Integration != "pi" || plan.Mutates {
		t.Fatalf("plan = %+v, want dry-run non-mutating pi plan", plan)
	}
	if plan.ConfigPath != settingsPath || plan.ExtensionPath != extensionPath {
		t.Fatalf("plan paths = %+v, want settings and extension paths", plan)
	}
	if plan.Protocol != "pi_extension" || plan.ConfigFormat != "json" || plan.ConfigAction != "add_extension_path" {
		t.Fatalf("protocol/config fields = %+v, want Pi extension settings patch", plan)
	}
	if plan.ServerURL != "http://127.0.0.1:8719" {
		t.Fatalf("server_url = %q, want local goncho-server URL", plan.ServerURL)
	}
	for _, want := range []string{`"extensions"`, extensionPath} {
		if !strings.Contains(plan.ConfigPatch, want) {
			t.Fatalf("config patch = %q, missing %q", plan.ConfigPatch, want)
		}
	}
	for _, want := range []string{"index.ts", "security.ts"} {
		if !slices.Contains(plan.ExtensionFiles, filepath.Join(extensionPath, want)) {
			t.Fatalf("extension files = %v, missing %s", plan.ExtensionFiles, want)
		}
	}
	if !slices.Contains(plan.GeneratedHookEvents, "prompt") || !slices.Contains(plan.GeneratedHookEvents, "pre_tool_use") || !slices.Contains(plan.GeneratedHookEvents, "session_end") {
		t.Fatalf("generated hook events = %v, want Pi-mappable host hook events", plan.GeneratedHookEvents)
	}
	for _, event := range plan.GeneratedHookEvents {
		if !slices.Contains(hostHookEventNames(), event) {
			t.Fatalf("generated hook event %q is not backed by HostHookEventSchemas", event)
		}
	}
}

func TestRunConnectCodexDryRunPrintsMCPConfigPatchWithoutMutating(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".codex", "config.toml")
	var stdout bytes.Buffer

	err := run(context.Background(), config{
		Command:    "connect",
		Connector:  "codex",
		DryRun:     true,
		ConfigPath: configPath,
		ServerAddr: "127.0.0.1:8719",
		Stdout:     &stdout,
	})
	if err != nil {
		t.Fatalf("run connect codex dry-run: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(configPath)); !os.IsNotExist(err) {
		t.Fatalf("dry-run created codex config dir or unexpected stat error: %v", err)
	}

	var plan connectPlan
	if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil {
		t.Fatalf("decode plan: %v\n%s", err, stdout.String())
	}
	if plan.Status != "dry_run" || plan.Integration != "codex" || plan.Mutates {
		t.Fatalf("plan = %+v, want dry-run non-mutating codex plan", plan)
	}
	if plan.ConfigPath != configPath {
		t.Fatalf("config_path = %q, want %q", plan.ConfigPath, configPath)
	}
	if plan.Protocol != "mcp" || plan.ConfigFormat != "toml" || plan.ConfigAction != "append_or_replace_mcp_server" {
		t.Fatalf("protocol/config fields = %+v, want MCP TOML append-or-replace", plan)
	}
	for _, want := range []string{`[mcp_servers.goncho]`, `command = "goncho-server"`, `args = ["serve", "-addr", "127.0.0.1:8719"]`} {
		if !strings.Contains(plan.ConfigPatch, want) {
			t.Fatalf("config patch = %q, missing %q", plan.ConfigPatch, want)
		}
	}
	if !slices.Contains(plan.GeneratedHookEvents, "prompt") || !slices.Contains(plan.GeneratedHookEvents, "tool_failure") {
		t.Fatalf("generated hook events = %v, want mappable host hook events", plan.GeneratedHookEvents)
	}
	for _, event := range plan.GeneratedHookEvents {
		if !slices.Contains(hostHookEventNames(), event) {
			t.Fatalf("generated hook event %q is not backed by HostHookEventSchemas", event)
		}
	}
	golden, err := os.ReadFile(filepath.Join("testdata", "codex_mcp_config.toml"))
	if err != nil {
		t.Fatalf("read golden config patch: %v", err)
	}
	if plan.ConfigPatch != string(golden) {
		t.Fatalf("config patch mismatch\ngot:\n%s\nwant:\n%s", plan.ConfigPatch, string(golden))
	}
}

func TestRunConnectGormesDryRunPrintsPlanWithoutMutating(t *testing.T) {
	dir := t.TempDir()
	profilesDir := filepath.Join(dir, ".gormes", "profiles")
	var stdout bytes.Buffer

	err := run(context.Background(), config{
		Command:           "connect",
		Connector:         "gormes",
		DryRun:            true,
		ProfilesDirectory: profilesDir,
		ProfileID:         "mineru",
		WorkspaceID:       "gormes-prod",
		ObserverID:        "gormes",
		Stdout:            &stdout,
	})
	if err != nil {
		t.Fatalf("run connect gormes dry-run: %v", err)
	}
	if _, err := os.Stat(profilesDir); !os.IsNotExist(err) {
		t.Fatalf("dry-run created profiles dir or unexpected stat error: %v", err)
	}

	var plan connectPlan
	if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil {
		t.Fatalf("decode plan: %v\n%s", err, stdout.String())
	}
	wantProfileDir := filepath.Join(profilesDir, "mineru")
	if plan.Status != "dry_run" || plan.Integration != "gormes" || plan.Mutates {
		t.Fatalf("plan = %+v, want dry-run non-mutating gormes plan", plan)
	}
	if plan.ProfileDirectory != wantProfileDir || plan.DatabasePath != filepath.Join(wantProfileDir, "goncho.db") || plan.MemoryMarkdownPath != filepath.Join(wantProfileDir, "GONCHO_MEMORY.md") {
		t.Fatalf("plan paths = %+v, want derived profile-local paths", plan)
	}
	wantTools := []string{"goncho_context", "goncho_search", "goncho_recall", "goncho_remember", "goncho_review", "goncho_handoff"}
	if !slices.Equal(plan.ToolNames, wantTools) {
		t.Fatalf("tools = %v, want %v", plan.ToolNames, wantTools)
	}
	if !slices.Contains(plan.HookEvents, "prompt") || !slices.Contains(plan.HookEvents, "session_end") {
		t.Fatalf("hook events = %v, want host hook schema events", plan.HookEvents)
	}
}

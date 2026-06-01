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

func TestConnectCursorPlanPrintsMCPPatch(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "Cursor", "mcp.json")
	var stdout bytes.Buffer

	if err := run(context.Background(), config{Command: "connect", Connector: "cursor", Plan: true, ConfigPath: configPath, ServerAddr: "127.0.0.1:8720", Stdout: &stdout}); err != nil {
		t.Fatalf("connect cursor --plan: %v", err)
	}
	if _, err := os.Stat(filepath.Dir(configPath)); !os.IsNotExist(err) {
		t.Fatalf("connect cursor --plan created config dir or unexpected stat error: %v", err)
	}
	var plan connectPlan
	if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil {
		t.Fatalf("decode cursor plan: %v\n%s", err, stdout.String())
	}
	if plan.Status != "plan" || plan.Operation != "connect" || plan.Integration != "cursor" || plan.Mutates {
		t.Fatalf("cursor plan = %+v, want non-mutating connect plan", plan)
	}
	if plan.Protocol != "mcp" || plan.ConfigFormat != "json" || plan.ConfigAction != "append_or_replace_mcp_server" {
		t.Fatalf("cursor config fields = %+v", plan)
	}
	for _, want := range []string{`"mcpServers"`, `"goncho"`, `"goncho-server"`, `"127.0.0.1:8720"`} {
		if !strings.Contains(plan.ConfigPatch, want) {
			t.Fatalf("cursor config patch = %q missing %q", plan.ConfigPatch, want)
		}
	}
}

func TestConnectGeminiCLIPlanPrintsMCPPatch(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".gemini", "settings.json")
	var stdout bytes.Buffer

	if err := run(context.Background(), config{Command: "connect", Connector: "gemini-cli", Plan: true, ConfigPath: configPath, ServerAddr: "127.0.0.1:8721", Stdout: &stdout}); err != nil {
		t.Fatalf("connect gemini-cli --plan: %v", err)
	}
	var plan connectPlan
	if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil {
		t.Fatalf("decode gemini plan: %v\n%s", err, stdout.String())
	}
	if plan.Status != "plan" || plan.Integration != "gemini-cli" || plan.Mutates || plan.ConfigPath != configPath {
		t.Fatalf("gemini plan = %+v", plan)
	}
	if plan.Protocol != "mcp" || plan.ConfigFormat != "json" || plan.ConfigAction != "append_or_replace_mcp_server" {
		t.Fatalf("gemini config fields = %+v", plan)
	}
	for _, want := range []string{`"mcpServers"`, `"goncho"`, `"command"`, `"goncho-server"`, `"127.0.0.1:8721"`} {
		if !strings.Contains(plan.ConfigPatch, want) {
			t.Fatalf("gemini config patch = %q missing %q", plan.ConfigPatch, want)
		}
	}
}

func TestConnectHermesPlanNamesGormesHandoff(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".hermes", "config.yaml")
	var stdout bytes.Buffer

	if err := run(context.Background(), config{Command: "connect", Connector: "hermes", Plan: true, ConfigPath: configPath, ServerAddr: "127.0.0.1:8722", Stdout: &stdout}); err != nil {
		t.Fatalf("connect hermes --plan: %v", err)
	}
	var plan connectPlan
	if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil {
		t.Fatalf("decode hermes plan: %v\n%s", err, stdout.String())
	}
	joinedSteps := strings.Join(plan.RecommendedNextStep, "\n")
	if plan.Status != "plan" || plan.Integration != "hermes" || plan.Mutates || plan.ConfigFormat != "yaml" {
		t.Fatalf("hermes plan = %+v", plan)
	}
	for _, want := range []string{"Gormes", "Hermes", "handoff"} {
		if !strings.Contains(joinedSteps, want) {
			t.Fatalf("hermes next steps = %v missing %q", plan.RecommendedNextStep, want)
		}
	}
	if !strings.Contains(plan.ConfigPatch, "goncho-server") || !strings.Contains(plan.ConfigPatch, "127.0.0.1:8722") {
		t.Fatalf("hermes config patch = %q", plan.ConfigPatch)
	}
}

func TestConnectOpenCodePlanUsesTopLevelMCPShape(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "opencode.json")
	var stdout bytes.Buffer

	if err := run(context.Background(), config{Command: "connect", Connector: "opencode", Plan: true, ConfigPath: configPath, ServerAddr: "127.0.0.1:8723", Stdout: &stdout}); err != nil {
		t.Fatalf("connect opencode --plan: %v", err)
	}
	var plan connectPlan
	if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil {
		t.Fatalf("decode opencode plan: %v\n%s", err, stdout.String())
	}
	if plan.Status != "plan" || plan.Integration != "opencode" || plan.Mutates || plan.ConfigFormat != "json" || plan.ConfigAction != "append_or_replace_top_level_mcp" {
		t.Fatalf("opencode plan = %+v", plan)
	}
	for _, want := range []string{`"mcp"`, `"type": "local"`, `"command"`, `"enabled": true`, `"127.0.0.1:8723"`} {
		if !strings.Contains(plan.ConfigPatch, want) {
			t.Fatalf("opencode config patch = %q missing %q", plan.ConfigPatch, want)
		}
	}
	if !slices.Contains(plan.GeneratedHookEvents, "prompt") || !slices.Contains(plan.GeneratedHookEvents, "tool_failure") {
		t.Fatalf("opencode generated hooks = %v", plan.GeneratedHookEvents)
	}
}

func TestRemovePlansMirrorConnectPlans(t *testing.T) {
	connectors := []string{"codex", "pi", "gormes", "cursor", "gemini-cli", "hermes", "opencode"}
	for _, connector := range connectors {
		t.Run(connector, func(t *testing.T) {
			var stdout bytes.Buffer
			cfg := config{Command: "remove", Connector: connector, Plan: true, ConfigPath: filepath.Join(t.TempDir(), connector+".conf"), ExtensionPath: filepath.Join(t.TempDir(), "goncho-extension"), DatabasePath: filepath.Join(t.TempDir(), "goncho.db"), ServerAddr: "127.0.0.1:8724", Stdout: &stdout}
			if err := run(context.Background(), cfg); err != nil {
				t.Fatalf("remove %s --plan: %v", connector, err)
			}
			var plan connectPlan
			if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil {
				t.Fatalf("decode remove plan: %v\n%s", err, stdout.String())
			}
			if plan.Status != "plan" || plan.Operation != "remove" || plan.Integration != connector || plan.Mutates {
				t.Fatalf("remove plan = %+v", plan)
			}
			if !strings.Contains(plan.ConfigAction, "remove") || strings.TrimSpace(plan.ConfigPatch) == "" {
				t.Fatalf("remove plan action/patch = %q/%q", plan.ConfigAction, plan.ConfigPatch)
			}
		})
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

func TestRunEmbeddingsReindexPlanReportsPreviewWithoutMutation(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "goncho.db")
	store, err := memory.OpenSqlite(dbPath, 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	if err := goncho.RunMigrations(store.DB()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	svc := goncho.NewService(store.DB(), goncho.Config{WorkspaceID: "default", ObserverPeerID: "gormes"}, nil)
	if _, err := svc.Conclude(ctx, goncho.ConcludeParams{Peer: "user-reindex", Conclusion: "Preview-only reindex candidate."}); err != nil {
		t.Fatalf("Conclude: %v", err)
	}
	if err := store.Close(ctx); err != nil {
		t.Fatalf("close store: %v", err)
	}

	var stdout bytes.Buffer
	if err := run(ctx, config{Command: "embeddings", Plan: true, DatabasePath: dbPath, Stdout: &stdout}); err != nil {
		t.Fatalf("run embeddings reindex --plan: %v", err)
	}
	var preview goncho.ReindexPreviewResult
	if err := json.Unmarshal(stdout.Bytes(), &preview); err != nil {
		t.Fatalf("decode reindex preview: %v\n%s", err, stdout.String())
	}
	if preview.Status != "ok" || preview.Mutates || preview.Total != 1 || preview.NotIndexed != 1 {
		t.Fatalf("preview = %+v, want non-mutating one missing vector", preview)
	}
}

func TestRunEmbeddingsDiagnoseReportsLocalVectorIndexHealth(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "goncho.db")
	store, err := memory.OpenSqlite(dbPath, 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	if err := goncho.RunMigrations(store.DB()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	svc := goncho.NewService(store.DB(), goncho.Config{WorkspaceID: "default", ObserverPeerID: "gormes"}, nil)
	if _, err := svc.Conclude(ctx, goncho.ConcludeParams{Peer: "user-diagnose", Conclusion: "Diagnose missing local vector row."}); err != nil {
		t.Fatalf("Conclude: %v", err)
	}
	if err := store.Close(ctx); err != nil {
		t.Fatalf("close store: %v", err)
	}

	var stdout bytes.Buffer
	if err := run(ctx, config{Command: "embeddings", Connector: "diagnose", DatabasePath: dbPath, EmbeddingIndexPath: filepath.Join(dir, "vectors.json"), Stdout: &stdout}); err != nil {
		t.Fatalf("run embeddings diagnose: %v", err)
	}
	var report goncho.EmbeddingDiagnosticsReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode diagnostics: %v\n%s", err, stdout.String())
	}
	if report.Status != "degraded" || report.Mutates || report.Preview.NotIndexed != 1 || report.VectorIndex.Path == "" {
		t.Fatalf("diagnostics = %+v, want degraded non-mutating missing-vector report", report)
	}
}

func TestRunEmbeddingsReindexApplyPopulatesLocalIndex(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "goncho.db")
	indexPath := filepath.Join(dir, "vectors.json")
	store, err := memory.OpenSqlite(dbPath, 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	if err := goncho.RunMigrations(store.DB()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	svc := goncho.NewService(store.DB(), goncho.Config{WorkspaceID: "default", ObserverPeerID: "gormes"}, nil)
	if _, err := svc.Conclude(ctx, goncho.ConcludeParams{Peer: "user-apply", Conclusion: "Apply reindex writes a local vector row."}); err != nil {
		t.Fatalf("Conclude: %v", err)
	}
	if err := store.Close(ctx); err != nil {
		t.Fatalf("close store: %v", err)
	}

	var stdout bytes.Buffer
	if err := run(ctx, config{Command: "embeddings", Connector: "reindex", Apply: true, DatabasePath: dbPath, EmbeddingIndexPath: indexPath, Stdout: &stdout}); err != nil {
		t.Fatalf("run embeddings reindex --apply: %v", err)
	}
	var result goncho.ReindexResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode reindex result: %v\n%s", err, stdout.String())
	}
	if !result.Mutates || result.Indexed != 1 || result.NotIndexed != 0 || result.Fresh != 1 {
		t.Fatalf("result = %+v, want one indexed fresh vector", result)
	}
	if _, err := os.Stat(indexPath); err != nil {
		t.Fatalf("index file not written: %v", err)
	}
}

func TestExportCommandWritesPortableManifest(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "goncho.db")
	outPath := filepath.Join(dir, "export.jsonl")
	store, err := memory.OpenSqlite(dbPath, 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	if err := goncho.RunMigrations(store.DB()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	svc := goncho.NewService(store.DB(), goncho.Config{WorkspaceID: "gormes", ObserverPeerID: "gormes"}, nil)
	if _, err := svc.Conclude(ctx, goncho.ConcludeParams{Peer: "user-export", Conclusion: "Portable export manifest memory."}); err != nil {
		t.Fatalf("Conclude: %v", err)
	}
	if err := store.Close(ctx); err != nil {
		t.Fatalf("close store: %v", err)
	}
	var stdout bytes.Buffer
	if err := run(ctx, config{Command: "export", DatabasePath: dbPath, IOPath: outPath, Format: "jsonl", Stdout: &stdout}); err != nil {
		t.Fatalf("run export: %v", err)
	}
	var manifest goncho.PortableExportManifest
	if err := json.Unmarshal(stdout.Bytes(), &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	if manifest.SchemaVersion != goncho.PortableExportSchemaVersion || manifest.Counts["conclusions"] == 0 || manifest.Checksum == "" {
		t.Fatalf("manifest = %+v, want portable manifest with conclusion count", manifest)
	}
	if info, err := os.Stat(outPath); err != nil || info.Size() == 0 {
		t.Fatalf("export file stat=%+v err=%v, want non-empty file", info, err)
	}
}

func TestImportPreviewReportsConflictsWithoutWrites(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "goncho.db")
	exportPath := filepath.Join(dir, "export.jsonl")
	store, err := memory.OpenSqlite(dbPath, 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	if err := goncho.RunMigrations(store.DB()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	svc := goncho.NewService(store.DB(), goncho.Config{WorkspaceID: "gormes", ObserverPeerID: "gormes"}, nil)
	if _, err := svc.Conclude(ctx, goncho.ConcludeParams{Peer: "user-import", Conclusion: "Portable import conflict memory."}); err != nil {
		t.Fatalf("Conclude: %v", err)
	}
	exported, err := svc.ExportPortableJSONL(ctx, goncho.PortableExportParams{})
	if err != nil {
		t.Fatalf("ExportPortableJSONL: %v", err)
	}
	if err := os.WriteFile(exportPath, exported.JSONL, 0o600); err != nil {
		t.Fatalf("write export: %v", err)
	}
	if err := store.Close(ctx); err != nil {
		t.Fatalf("close store: %v", err)
	}
	var stdout bytes.Buffer
	if err := run(ctx, config{Command: "import", Connector: "preview", DatabasePath: dbPath, IOPath: exportPath, Stdout: &stdout}); err != nil {
		t.Fatalf("run import preview: %v", err)
	}
	var preview goncho.PortableImportPreview
	if err := json.Unmarshal(stdout.Bytes(), &preview); err != nil {
		t.Fatalf("decode preview: %v", err)
	}
	if preview.Mutates || preview.SafeToApply || len(preview.Conflicts) == 0 {
		t.Fatalf("preview = %+v, want non-mutating conflict preview", preview)
	}
}

func TestImportApplyRequiresConfirm(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "goncho.db")
	inPath := filepath.Join(dir, "empty.jsonl")
	if err := os.WriteFile(inPath, []byte("{}\n"), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}
	var stdout bytes.Buffer
	err := run(context.Background(), config{Command: "import", Connector: "apply", DatabasePath: dbPath, IOPath: inPath, Stdout: &stdout})
	if err == nil || !strings.Contains(err.Error(), "--confirm APPLY") {
		t.Fatalf("err = %v, want confirm requirement", err)
	}
}

func TestRetentionReportCommandIsReadOnly(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "goncho.db")
	store, err := memory.OpenSqlite(dbPath, 0, nil)
	if err != nil {
		t.Fatalf("OpenSqlite: %v", err)
	}
	if err := goncho.RunMigrations(store.DB()); err != nil {
		t.Fatalf("RunMigrations: %v", err)
	}
	svc := goncho.NewService(store.DB(), goncho.Config{WorkspaceID: "gormes", ObserverPeerID: "gormes"}, nil)
	if _, err := svc.Conclude(ctx, goncho.ConcludeParams{Peer: "user-report", Conclusion: "Retention report password token candidate."}); err != nil {
		t.Fatalf("Conclude: %v", err)
	}
	if err := store.Close(ctx); err != nil {
		t.Fatalf("close store: %v", err)
	}

	var stdout bytes.Buffer
	if err := run(ctx, config{Command: "report", Connector: "retention", ReportJSON: true, DatabasePath: dbPath, Stdout: &stdout}); err != nil {
		t.Fatalf("run report retention --json: %v", err)
	}
	var report goncho.RetentionAccessReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode retention report: %v\n%s", err, stdout.String())
	}
	if report.Mutates || report.Status != "ok" || report.Counts["high_risk"] == 0 {
		t.Fatalf("report = %+v, want read-only high-risk report", report)
	}
}

func TestRunQuickstartPlanPrintsNonMutatingGuideWithStepsAndNextCommands(t *testing.T) {
	dir := t.TempDir()
	var stdout bytes.Buffer

	if err := run(context.Background(), config{
		Command:              "quickstart",
		Plan:                 true,
		QuickstartDBPath:     filepath.Join(dir, "goncho.db"),
		QuickstartServerAddr: "127.0.0.1:8765",
		Stdout:               &stdout,
	}); err != nil {
		t.Fatalf("run quickstart --plan: %v", err)
	}
	// Plan must not create any files.
	dbPath := filepath.Join(dir, "goncho.db")
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Fatalf("quickstart --plan created db at %s or unexpected stat error: %v", dbPath, err)
	}

	var plan quickstartPlan
	if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil {
		t.Fatalf("decode quickstart plan: %v\n%s", err, stdout.String())
	}
	if plan.Status != "plan" || plan.Mutates {
		t.Fatalf("quickstart plan = %+v, want non-mutating plan", plan)
	}
	if plan.DatabasePath == "" {
		t.Fatalf("quickstart plan missing database_path")
	}
	if len(plan.Steps) == 0 {
		t.Fatalf("quickstart plan has zero steps")
	}
	// Each step must have a command and description.
	for i, step := range plan.Steps {
		if step.Command == "" || step.Description == "" {
			t.Fatalf("quickstart step %d missing command or description: %+v", i, step)
		}
	}
	// Must include viewer URL, demo write, recall, and context proof.
	if plan.ViewerURL == "" {
		t.Fatalf("quickstart plan missing viewer_url")
	}
	if !strings.Contains(plan.DemoProof, "conclusion") || !strings.Contains(plan.DemoProof, "viewer") || !strings.Contains(plan.DemoProof, "context") {
		t.Fatalf("quickstart plan demo_proof = %q, want conclusion/viewer/context commands", plan.DemoProof)
	}
	// Must include recommended next steps for connectors.
	if len(plan.NextSteps) == 0 {
		t.Fatalf("quickstart plan missing next_steps")
	}
}

func TestConnectCodexPlanIncludesHookBundleManifest(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".codex", "config.toml")
	var stdout bytes.Buffer

	if err := run(context.Background(), config{Command: "connect", Connector: "codex", Plan: true, ConfigPath: configPath, ServerAddr: "127.0.0.1:8725", Stdout: &stdout}); err != nil {
		t.Fatalf("connect codex --plan: %v", err)
	}
	var plan connectPlan
	if err := json.Unmarshal(stdout.Bytes(), &plan); err != nil {
		t.Fatalf("decode plan: %v\n%s", err, stdout.String())
	}
	if len(plan.HookBundles) == 0 {
		t.Fatalf("hook_bundles empty in plan: %+v", plan)
	}
	events := make([]string, 0, len(plan.HookBundles))
	for _, bundle := range plan.HookBundles {
		events = append(events, bundle.Event)
		if bundle.Host != "codex" || bundle.InstallStatus != "plan_only" || bundle.RedactionClass == "" || bundle.PayloadSchema == nil || len(bundle.Command) == 0 || bundle.OutputPath == "" {
			t.Fatalf("hook bundle = %+v, want complete codex plan-only manifest", bundle)
		}
	}
	for _, want := range []string{"prompt", "pre_tool_use", "post_tool_use", "tool_failure", "session_end"} {
		if !slices.Contains(events, want) {
			t.Fatalf("hook bundle events = %v, missing %s", events, want)
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

package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	gormes "github.com/TrebuchetDynamics/goncho/integration/gormes"
	goncho "github.com/TrebuchetDynamics/goncho/service"
	_ "github.com/ncruces/go-sqlite3/driver"
)

type config struct {
	Command               string
	Connector             string
	DryRun                bool
	Plan                  bool
	Apply                 bool
	ProfilesDirectory     string
	ProfileID             string
	WorkspaceID           string
	ObserverID            string
	DatabasePath          string
	ConfigPath            string
	ExtensionPath         string
	ServerAddr            string
	PreferencesConfigPath string
	PreferenceUpdates     map[string]string
	WatchRoots            []string
	IncludeGlobs          []string
	ExcludeGlobs          []string
	VersionJSON           bool
	DoctorJSON            bool
	UpgradeJSON           bool
	CurrentVersion        string
	LatestVersion         string
	SchemaFingerprintJSON bool
	Stdout                io.Writer
	Stderr                io.Writer
}

type connectPlan struct {
	Status              string   `json:"status"`
	Operation           string   `json:"operation"`
	Integration         string   `json:"integration"`
	Mutates             bool     `json:"mutates"`
	WorkspaceID         string   `json:"workspace_id,omitempty"`
	ObserverID          string   `json:"observer_id,omitempty"`
	ProfileID           string   `json:"profile_id,omitempty"`
	ProfilesDirectory   string   `json:"profiles_directory,omitempty"`
	ProfileDirectory    string   `json:"profile_directory,omitempty"`
	DatabasePath        string   `json:"database_path,omitempty"`
	MemoryMarkdownPath  string   `json:"memory_markdown_path,omitempty"`
	ToolNames           []string `json:"tool_names,omitempty"`
	HookEvents          []string `json:"hook_events,omitempty"`
	MCPCommand          []string `json:"mcp_command,omitempty"`
	ServerURL           string   `json:"server_url,omitempty"`
	ConfigPath          string   `json:"config_path,omitempty"`
	ExtensionPath       string   `json:"extension_path,omitempty"`
	ExtensionFiles      []string `json:"extension_files,omitempty"`
	Protocol            string   `json:"protocol,omitempty"`
	ConfigFormat        string   `json:"config_format,omitempty"`
	ConfigAction        string   `json:"config_action,omitempty"`
	ConfigPatch         string   `json:"config_patch,omitempty"`
	GeneratedHookEvents []string `json:"generated_hook_events,omitempty"`
	WatchRoots          []string `json:"watch_roots,omitempty"`
	IncludeGlobs        []string `json:"include_globs,omitempty"`
	ExcludeGlobs        []string `json:"exclude_globs,omitempty"`
	RecommendedNextStep []string `json:"recommended_next_steps"`
}

type operatorPreferences struct {
	DBPath              string `json:"db_path"`
	WorkspaceID         string `json:"workspace_id"`
	ProfileID           string `json:"profile_id"`
	RedactionPolicy     string `json:"redaction_policy"`
	ConnectorPermission string `json:"connector_permission"`
	BindAddr            string `json:"bind_addr"`
}

type preferencesReport struct {
	Status      string              `json:"status"`
	Mutates     bool                `json:"mutates"`
	ConfigPath  string              `json:"config_path"`
	Preferences operatorPreferences `json:"preferences"`
}

type versionReport struct {
	Service         string `json:"service"`
	ModuleVersion   string `json:"module_version"`
	GitCommit       string `json:"git_commit,omitempty"`
	DBSchemaVersion string `json:"db_schema_version"`
	PublicToolCount int    `json:"public_tool_count"`
	Mutates         bool   `json:"mutates"`
}

type schemaFingerprintReport struct {
	Service         string   `json:"service"`
	DBSchemaVersion string   `json:"db_schema_version"`
	PublicToolNames []string `json:"public_tool_names"`
	PublicToolCount int      `json:"public_tool_count"`
	HostHookEvents  []string `json:"host_hook_events"`
	Fingerprint     string   `json:"fingerprint"`
	Mutates         bool     `json:"mutates"`
}

type upgradeCheckReport struct {
	Status          string   `json:"status"`
	CurrentVersion  string   `json:"current_version"`
	LatestVersion   string   `json:"latest_version,omitempty"`
	UpdateAvailable bool     `json:"update_available"`
	Mutates         bool     `json:"mutates"`
	NextSteps       []string `json:"next_steps,omitempty"`
}

type doctorReport struct {
	Status  string        `json:"status"`
	Mutates bool          `json:"mutates"`
	DBPath  string        `json:"db_path,omitempty"`
	Checks  []doctorCheck `json:"checks"`
}

type doctorCheck struct {
	Name        string   `json:"name"`
	Status      string   `json:"status"`
	Message     string   `json:"message,omitempty"`
	Suggestions []string `json:"suggestions,omitempty"`
}

func (r doctorReport) CheckByName(name string) (doctorCheck, bool) {
	for _, check := range r.Checks {
		if check.Name == name {
			return check, true
		}
	}
	return doctorCheck{}, false
}

func main() {
	cfg, err := parseConfig(os.Args[1:], os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := run(context.Background(), cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parseConfig(args []string, stdout, stderr io.Writer) (config, error) {
	cfg := config{Stdout: stdout, Stderr: stderr}
	if len(args) == 0 {
		return cfg, errors.New("goncho: command is required")
	}
	cfg.Command = strings.TrimSpace(args[0])
	args = args[1:]
	switch cfg.Command {
	case "connect", "remove":
		if len(args) == 0 || strings.HasPrefix(args[0], "-") {
			return cfg, fmt.Errorf("goncho %s: connector is required", cfg.Command)
		}
		cfg.Connector = strings.TrimSpace(args[0])
		args = args[1:]
		fs := flag.NewFlagSet("goncho "+cfg.Command+" "+cfg.Connector, flag.ContinueOnError)
		fs.SetOutput(stderr)
		fs.BoolVar(&cfg.DryRun, "dry-run", false, "print planned changes without mutating files")
		fs.BoolVar(&cfg.Plan, "plan", false, "print a reversible non-mutating plan")
		fs.BoolVar(&cfg.Apply, "apply", false, "apply connector changes")
		fs.StringVar(&cfg.ProfilesDirectory, "profiles-dir", "", "Gormes profiles directory")
		fs.StringVar(&cfg.ProfileID, "profile", "", "Gormes profile ID")
		fs.StringVar(&cfg.WorkspaceID, "workspace", "", "Goncho workspace ID")
		fs.StringVar(&cfg.ObserverID, "observer", "", "Goncho observer peer ID")
		fs.StringVar(&cfg.DatabasePath, "db", "", "explicit Goncho SQLite database path")
		fs.StringVar(&cfg.ConfigPath, "config", "", "external host config path for connector plans")
		fs.StringVar(&cfg.ExtensionPath, "extension", "", "Pi extension directory for connector plans")
		watchRoots := stringListFlag{}
		includeGlobs := stringListFlag{}
		excludeGlobs := stringListFlag{}
		fs.StringVar(&cfg.ServerAddr, "addr", "", "goncho-server listen address for connector plans")
		fs.Var(&watchRoots, "watch-root", "filesystem watcher root to observe; repeatable")
		fs.Var(&includeGlobs, "include", "filesystem watcher include glob; repeatable and required for filesystem-watcher")
		fs.Var(&excludeGlobs, "exclude", "filesystem watcher exclude glob; repeatable")
		if err := fs.Parse(args); err != nil {
			return cfg, err
		}
		cfg.WatchRoots = []string(watchRoots)
		cfg.IncludeGlobs = []string(includeGlobs)
		cfg.ExcludeGlobs = []string(excludeGlobs)
		return cfg, nil
	case "schema-fingerprint":
		fs := flag.NewFlagSet("goncho schema-fingerprint", flag.ContinueOnError)
		fs.SetOutput(stderr)
		fs.BoolVar(&cfg.SchemaFingerprintJSON, "json", false, "print schema fingerprint as JSON")
		if err := fs.Parse(args); err != nil {
			return cfg, err
		}
		return cfg, nil
	case "upgrade-check":
		fs := flag.NewFlagSet("goncho upgrade-check", flag.ContinueOnError)
		fs.SetOutput(stderr)
		fs.BoolVar(&cfg.UpgradeJSON, "json", false, "print upgrade check as JSON")
		fs.StringVar(&cfg.CurrentVersion, "current", "", "current Goncho version; defaults to build metadata")
		fs.StringVar(&cfg.LatestVersion, "latest", "", "latest known Goncho version from a trusted release source")
		if err := fs.Parse(args); err != nil {
			return cfg, err
		}
		return cfg, nil
	case "doctor":
		fs := flag.NewFlagSet("goncho doctor", flag.ContinueOnError)
		fs.SetOutput(stderr)
		fs.BoolVar(&cfg.DoctorJSON, "json", false, "print doctor report as JSON")
		fs.StringVar(&cfg.DatabasePath, "db", "", "Goncho SQLite database path to inspect without creating it")
		fs.StringVar(&cfg.PreferencesConfigPath, "config", "", "Goncho preferences JSON path")
		if err := fs.Parse(args); err != nil {
			return cfg, err
		}
		return cfg, nil
	case "version":
		fs := flag.NewFlagSet("goncho version", flag.ContinueOnError)
		fs.SetOutput(stderr)
		fs.BoolVar(&cfg.VersionJSON, "json", false, "print version metadata as JSON")
		if err := fs.Parse(args); err != nil {
			return cfg, err
		}
		return cfg, nil
	case "preferences":
		updates := preferenceUpdateFlag{}
		fs := flag.NewFlagSet("goncho preferences", flag.ContinueOnError)
		fs.SetOutput(stderr)
		fs.StringVar(&cfg.PreferencesConfigPath, "config", "", "Goncho preferences JSON path")
		fs.Var(&updates, "set", "set preference key=value")
		if err := fs.Parse(args); err != nil {
			return cfg, err
		}
		cfg.PreferenceUpdates = map[string]string(updates)
		return cfg, nil
	default:
		return cfg, fmt.Errorf("goncho: unknown command %q (want connect, remove, preferences, doctor, upgrade-check, schema-fingerprint, or version)", cfg.Command)
	}
}

func run(ctx context.Context, cfg config) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if cfg.Stdout == nil {
		cfg.Stdout = os.Stdout
	}
	switch cfg.Command {
	case "connect":
		if cfg.Apply {
			return fmt.Errorf("goncho connect %s: --apply is not implemented yet; use --plan or --dry-run", cfg.Connector)
		}
		if !cfg.DryRun && !cfg.Plan {
			return fmt.Errorf("goncho connect %s: pass --plan or --dry-run to inspect the non-mutating plan", cfg.Connector)
		}
		plan, err := buildConnectPlan(cfg)
		if err != nil {
			return err
		}
		plan.Operation = "connect"
		plan.Status = planStatus(cfg)
		return json.NewEncoder(cfg.Stdout).Encode(plan)
	case "remove":
		if cfg.Apply {
			return fmt.Errorf("goncho remove %s: --apply is not implemented yet; use --plan or --dry-run", cfg.Connector)
		}
		if !cfg.DryRun && !cfg.Plan {
			return fmt.Errorf("goncho remove %s: pass --plan or --dry-run to inspect the non-mutating plan", cfg.Connector)
		}
		plan, err := buildRemovePlan(cfg)
		if err != nil {
			return err
		}
		plan.Operation = "remove"
		plan.Status = planStatus(cfg)
		return json.NewEncoder(cfg.Stdout).Encode(plan)
	case "preferences":
		return runPreferences(ctx, cfg)
	case "doctor":
		return runDoctor(ctx, cfg)
	case "upgrade-check":
		return runUpgradeCheck(ctx, cfg)
	case "schema-fingerprint":
		return runSchemaFingerprint(ctx, cfg)
	case "version":
		return runVersion(ctx, cfg)
	default:
		return fmt.Errorf("goncho: unknown command %q", cfg.Command)
	}
}

func planStatus(cfg config) string {
	if cfg.Plan {
		return "plan"
	}
	return "dry_run"
}

func buildConnectPlan(cfg config) (connectPlan, error) {
	switch cfg.Connector {
	case "gormes":
		return buildGormesConnectPlan(cfg)
	case "codex":
		return buildCodexConnectPlan(cfg)
	case "pi":
		return buildPiConnectPlan(cfg)
	case "filesystem-watcher":
		return buildFilesystemWatcherConnectPlan(cfg)
	default:
		return connectPlan{}, fmt.Errorf("goncho connect: unsupported connector %q", cfg.Connector)
	}
}

func buildRemovePlan(cfg config) (connectPlan, error) {
	switch cfg.Connector {
	case "codex":
		plan, err := buildCodexConnectPlan(cfg)
		if err != nil {
			return connectPlan{}, err
		}
		plan.ConfigAction = "remove_mcp_server"
		plan.ConfigPatch = "# Remove the Goncho MCP server block from Codex config:\n[mcp_servers.goncho]\n"
		plan.MCPCommand = nil
		plan.GeneratedHookEvents = nil
		plan.RecommendedNextStep = []string{"Remove the [mcp_servers.goncho] block from Codex config after confirming no active sessions use it.", "Stop goncho-server only after all connected hosts are disconnected."}
		return plan, nil
	case "pi":
		plan, err := buildPiConnectPlan(cfg)
		if err != nil {
			return connectPlan{}, err
		}
		plan.ConfigAction = "remove_extension_path"
		plan.ConfigPatch = piRemoveSettingsConfigPatch(plan.ExtensionPath)
		plan.ExtensionFiles = nil
		plan.GeneratedHookEvents = nil
		plan.RecommendedNextStep = []string{"Remove the Goncho extension path from Pi settings.", "Delete copied extension files only after export/review if they were locally modified."}
		return plan, nil
	case "gormes":
		plan, err := buildGormesConnectPlan(cfg)
		if err != nil {
			return connectPlan{}, err
		}
		plan.ConfigAction = "remove_tools_and_hooks"
		plan.ConfigPatch = "Remove Goncho public tools and host hook forwarding from the Gormes profile configuration; leave the local Goncho DB untouched."
		plan.RecommendedNextStep = []string{"Disable Gormes hook forwarding before deleting any local adapter files.", "Keep the Goncho SQLite DB and markdown mirror unless the operator explicitly archives or deletes them."}
		return plan, nil
	default:
		return connectPlan{}, fmt.Errorf("goncho remove: unsupported connector %q", cfg.Connector)
	}
}

func runSchemaFingerprint(ctx context.Context, cfg config) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if cfg.Stdout == nil {
		cfg.Stdout = os.Stdout
	}
	report := schemaFingerprintReport{
		Service:         "goncho",
		DBSchemaVersion: goncho.GonchoSQLiteSchemaVersion,
		PublicToolNames: publicToolNames(),
		PublicToolCount: len(publicToolNames()),
		HostHookEvents:  hostHookEventNames(),
		Mutates:         false,
	}
	payload := struct {
		DBSchemaVersion string   `json:"db_schema_version"`
		PublicToolNames []string `json:"public_tool_names"`
		HostHookEvents  []string `json:"host_hook_events"`
	}{report.DBSchemaVersion, report.PublicToolNames, report.HostHookEvents}
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("goncho schema-fingerprint: encode payload: %w", err)
	}
	sum := sha256.Sum256(raw)
	report.Fingerprint = "sha256:" + hex.EncodeToString(sum[:])
	if cfg.SchemaFingerprintJSON {
		return json.NewEncoder(cfg.Stdout).Encode(report)
	}
	return json.NewEncoder(cfg.Stdout).Encode(report)
}

func runUpgradeCheck(ctx context.Context, cfg config) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if cfg.Stdout == nil {
		cfg.Stdout = os.Stdout
	}
	current := strings.TrimSpace(cfg.CurrentVersion)
	if current == "" {
		current = buildVersion()
	}
	latest := strings.TrimSpace(cfg.LatestVersion)
	report := upgradeCheckReport{Status: "unknown", CurrentVersion: current, LatestVersion: latest, Mutates: false}
	if latest == "" {
		report.NextSteps = []string{"Check GitHub releases or pkg.go.dev from a trusted network environment; upgrade-check does not mutate files."}
	} else if versionGreater(latest, current) {
		report.Status = "update_available"
		report.UpdateAvailable = true
		report.NextSteps = []string{"Review the release notes before upgrading.", "Run make release-smoke or project-specific smoke tests after changing the pinned version."}
	} else {
		report.Status = "current"
		report.NextSteps = []string{"No newer trusted release was supplied; keep using the pinned version."}
	}
	if cfg.UpgradeJSON {
		return json.NewEncoder(cfg.Stdout).Encode(report)
	}
	return json.NewEncoder(cfg.Stdout).Encode(report)
}

func versionGreater(a, b string) bool {
	ap := parseVersionParts(a)
	bp := parseVersionParts(b)
	for i := 0; i < len(ap) || i < len(bp); i++ {
		av, bv := 0, 0
		if i < len(ap) {
			av = ap[i]
		}
		if i < len(bp) {
			bv = bp[i]
		}
		if av != bv {
			return av > bv
		}
	}
	return false
}

func parseVersionParts(v string) []int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	v, _, _ = strings.Cut(v, "-")
	pieces := strings.Split(v, ".")
	out := make([]int, 0, len(pieces))
	for _, piece := range pieces {
		var n int
		for _, r := range piece {
			if r < '0' || r > '9' {
				break
			}
			n = n*10 + int(r-'0')
		}
		out = append(out, n)
	}
	return out
}

func runDoctor(ctx context.Context, cfg config) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if cfg.Stdout == nil {
		cfg.Stdout = os.Stdout
	}
	dbPath := strings.TrimSpace(cfg.DatabasePath)
	if dbPath == "" {
		if configPath, err := preferenceConfigPath(cfg.PreferencesConfigPath); err == nil {
			if prefs, err := readPreferences(configPath); err == nil {
				dbPath = strings.TrimSpace(prefs.DBPath)
			}
		}
	}
	report := doctorReport{Status: "ok", Mutates: false, DBPath: dbPath}
	addCheck := func(name string, err error, suggestions []string) {
		check := doctorCheck{Name: name, Status: "ok"}
		if err != nil {
			check.Status = "error"
			check.Message = err.Error()
			check.Suggestions = suggestions
			report.Status = "error"
		}
		report.Checks = append(report.Checks, check)
	}
	addCheck("db_path", checkDoctorDBPath(dbPath), []string{"Run goncho-server init -db <path> or set goncho preferences --set db_path=<path>.", "Use goncho connect <host> --plan before enabling hooks."})
	addCheck("migrations", checkDoctorMigrations(ctx, dbPath), []string{"Run goncho-server init -db <path> to create or migrate the local database."})
	addCheck("preferences", checkDoctorPreferences(cfg.PreferencesConfigPath), []string{"Run goncho preferences --config <path> --set db_path=<path> to write local operator preferences."})
	if len(publicToolNames()) == 0 {
		addCheck("public_tools", errors.New("no public Goncho tools registered"), []string{"Run go test ./cmd/goncho ./service and inspect public tool registration."})
	} else {
		addCheck("public_tools", nil, nil)
	}
	if cfg.DoctorJSON {
		return json.NewEncoder(cfg.Stdout).Encode(report)
	}
	return json.NewEncoder(cfg.Stdout).Encode(report)
}

func checkDoctorDBPath(dbPath string) error {
	if strings.TrimSpace(dbPath) == "" {
		return errors.New("db path is not configured")
	}
	info, err := os.Stat(dbPath)
	if err != nil {
		return fmt.Errorf("db path is not readable: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("db path %q is a directory", dbPath)
	}
	return nil
}

func checkDoctorMigrations(ctx context.Context, dbPath string) error {
	if err := checkDoctorDBPath(dbPath); err != nil {
		return err
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}
	defer db.Close()
	var name string
	err = db.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'goncho_observations'`).Scan(&name)
	if err != nil {
		return fmt.Errorf("goncho migrations missing or incomplete: %w", err)
	}
	return nil
}

func checkDoctorPreferences(path string) error {
	configPath, err := preferenceConfigPath(path)
	if err != nil {
		return err
	}
	if _, err := readPreferences(configPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func runVersion(ctx context.Context, cfg config) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if cfg.Stdout == nil {
		cfg.Stdout = os.Stdout
	}
	report := versionReport{
		Service:         "goncho",
		ModuleVersion:   buildVersion(),
		GitCommit:       buildGitCommit(),
		DBSchemaVersion: goncho.GonchoSQLiteSchemaVersion,
		PublicToolCount: len(publicToolNames()),
		Mutates:         false,
	}
	if cfg.VersionJSON {
		return json.NewEncoder(cfg.Stdout).Encode(report)
	}
	_, err := fmt.Fprintf(cfg.Stdout, "%s %s schema=%s tools=%d\n", report.Service, report.ModuleVersion, report.DBSchemaVersion, report.PublicToolCount)
	return err
}

func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || strings.TrimSpace(info.Main.Version) == "" {
		return "(devel)"
	}
	return info.Main.Version
}

func buildGitCommit() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			return setting.Value
		}
	}
	return ""
}

func publicToolNames() []string {
	return []string{"goncho_context", "goncho_search", "goncho_recall", "goncho_remember", "goncho_review", "goncho_handoff"}
}

func runPreferences(ctx context.Context, cfg config) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	configPath, err := preferenceConfigPath(cfg.PreferencesConfigPath)
	if err != nil {
		return err
	}
	prefs, _ := readPreferences(configPath)
	if prefs == (operatorPreferences{}) {
		prefs = defaultPreferences()
	}
	mutates := len(cfg.PreferenceUpdates) > 0
	if mutates {
		if err := applyPreferenceUpdates(&prefs, cfg.PreferenceUpdates); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
			return fmt.Errorf("goncho preferences: create config dir: %w", err)
		}
		raw, err := json.MarshalIndent(prefs, "", "  ")
		if err != nil {
			return fmt.Errorf("goncho preferences: encode config: %w", err)
		}
		raw = append(raw, '\n')
		if err := os.WriteFile(configPath, raw, 0o600); err != nil {
			return fmt.Errorf("goncho preferences: write config: %w", err)
		}
	}
	return json.NewEncoder(cfg.Stdout).Encode(preferencesReport{Status: "ok", Mutates: mutates, ConfigPath: configPath, Preferences: prefs})
}

func preferenceConfigPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path != "" {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "", errors.New("goncho preferences: --config is required when home directory cannot be resolved")
	}
	return filepath.Join(home, ".config", "goncho", "preferences.json"), nil
}

func readPreferences(path string) (operatorPreferences, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return operatorPreferences{}, err
	}
	var prefs operatorPreferences
	if err := json.Unmarshal(raw, &prefs); err != nil {
		return operatorPreferences{}, fmt.Errorf("goncho preferences: decode config: %w", err)
	}
	return prefs, nil
}

func defaultPreferences() operatorPreferences {
	return operatorPreferences{WorkspaceID: "default", RedactionPolicy: "standard", ConnectorPermission: "plan_only", BindAddr: "127.0.0.1:8765"}
}

func applyPreferenceUpdates(prefs *operatorPreferences, updates map[string]string) error {
	for key, value := range updates {
		value = strings.TrimSpace(value)
		switch strings.TrimSpace(key) {
		case "db_path":
			prefs.DBPath = value
		case "workspace_id":
			prefs.WorkspaceID = value
		case "profile_id":
			prefs.ProfileID = value
		case "redaction_policy":
			prefs.RedactionPolicy = value
		case "connector_permission":
			prefs.ConnectorPermission = value
		case "bind_addr":
			prefs.BindAddr = value
		default:
			return fmt.Errorf("goncho preferences: unknown key %q", key)
		}
	}
	return nil
}

type stringListFlag []string

func (f *stringListFlag) String() string {
	if f == nil {
		return ""
	}
	return strings.Join(*f, ",")
}

func (f *stringListFlag) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("value must not be empty")
	}
	*f = append(*f, value)
	return nil
}

func normalizeCLIStringList(values []string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

type preferenceUpdateFlag map[string]string

func (f *preferenceUpdateFlag) String() string {
	if f == nil {
		return ""
	}
	return fmt.Sprint(map[string]string(*f))
}

func (f *preferenceUpdateFlag) Set(value string) error {
	key, val, ok := strings.Cut(value, "=")
	if !ok || strings.TrimSpace(key) == "" {
		return fmt.Errorf("preference update must be key=value")
	}
	if *f == nil {
		*f = preferenceUpdateFlag{}
	}
	(*f)[strings.TrimSpace(key)] = val
	return nil
}

func buildGormesConnectPlan(cfg config) (connectPlan, error) {
	workspaceID := strings.TrimSpace(cfg.WorkspaceID)
	if workspaceID == "" {
		workspaceID = gormes.DefaultWorkspaceID
	}
	observerID := strings.TrimSpace(cfg.ObserverID)
	if observerID == "" {
		observerID = gormes.DefaultObserverID
	}
	profilesDir := strings.TrimSpace(cfg.ProfilesDirectory)
	profileID := strings.TrimSpace(cfg.ProfileID)
	databasePath := strings.TrimSpace(cfg.DatabasePath)
	if databasePath == "" {
		if profilesDir == "" || profileID == "" {
			return connectPlan{}, errors.New("goncho connect gormes: either --db or both --profiles-dir and --profile are required")
		}
		if err := validateGormesProfileID(profileID); err != nil {
			return connectPlan{}, err
		}
		databasePath = filepath.Join(profilesDir, profileID, "goncho.db")
	}
	profileDir := ""
	if profilesDir != "" && profileID != "" {
		if err := validateGormesProfileID(profileID); err != nil {
			return connectPlan{}, err
		}
		profileDir = filepath.Join(profilesDir, profileID)
	}
	memoryMarkdownPath := filepath.Join(filepath.Dir(databasePath), "GONCHO_MEMORY.md")
	return connectPlan{
		Status:             "dry_run",
		Integration:        "gormes",
		Mutates:            false,
		WorkspaceID:        workspaceID,
		ObserverID:         observerID,
		ProfileID:          profileID,
		ProfilesDirectory:  profilesDir,
		ProfileDirectory:   profileDir,
		DatabasePath:       databasePath,
		MemoryMarkdownPath: memoryMarkdownPath,
		ToolNames:          []string{"goncho_context", "goncho_search", "goncho_recall", "goncho_remember", "goncho_review", "goncho_handoff"},
		HookEvents:         hostHookEventNames(),
		MCPCommand:         []string{"goncho-server", "serve", "-db", databasePath},
		RecommendedNextStep: []string{
			"Review this plan with the Gormes host configuration owner.",
			"Run goncho-server init with the planned database path before enabling hooks.",
			"Register the listed Goncho public tools in the Gormes tool registry.",
		},
	}, nil
}

func buildFilesystemWatcherConnectPlan(cfg config) (connectPlan, error) {
	roots := normalizeCLIStringList(cfg.WatchRoots)
	include := normalizeCLIStringList(cfg.IncludeGlobs)
	exclude := normalizeCLIStringList(cfg.ExcludeGlobs)
	if len(roots) == 0 {
		return connectPlan{}, errors.New("goncho connect filesystem-watcher: at least one --watch-root is required")
	}
	if len(include) == 0 {
		return connectPlan{}, errors.New("goncho connect filesystem-watcher: at least one explicit --include glob is required")
	}
	if len(exclude) == 0 {
		exclude = []string{".git/**", "node_modules/**", "dist/**", "build/**", "coverage/**", "*.log", "*.lock"}
	}
	return connectPlan{
		Status:       "dry_run",
		Integration:  "filesystem-watcher",
		Mutates:      false,
		Protocol:     "local_service_api",
		ConfigFormat: "json",
		ConfigAction: "preview_import_changed_files",
		WatchRoots:   roots,
		IncludeGlobs: include,
		ExcludeGlobs: exclude,
		RecommendedNextStep: []string{
			"Run PreviewFilesystemWatcherImport with a sample changed-file batch and inspect skipped/importable counts.",
			"Only call ImportFilesystemWatcherChanges after include/exclude rules are reviewed by the workspace owner.",
			"Keep the watcher one-way: changed files become scoped observations; it must not mutate source files.",
		},
	}, nil
}

func buildPiConnectPlan(cfg config) (connectPlan, error) {
	settingsPath := strings.TrimSpace(cfg.ConfigPath)
	extensionPath := strings.TrimSpace(cfg.ExtensionPath)
	if settingsPath == "" || extensionPath == "" {
		home, err := os.UserHomeDir()
		if err != nil || strings.TrimSpace(home) == "" {
			return connectPlan{}, errors.New("goncho connect pi: --config and --extension are required when home directory cannot be resolved")
		}
		if settingsPath == "" {
			settingsPath = filepath.Join(home, ".pi", "agent", "settings.json")
		}
		if extensionPath == "" {
			extensionPath = filepath.Join(home, ".pi", "agent", "extensions", "goncho")
		}
	}
	serverAddr := strings.TrimSpace(cfg.ServerAddr)
	if serverAddr == "" {
		serverAddr = "127.0.0.1:8765"
	}
	serverURL := "http://" + serverAddr
	return connectPlan{
		Status:         "dry_run",
		Integration:    "pi",
		Mutates:        false,
		ConfigPath:     settingsPath,
		ExtensionPath:  extensionPath,
		ExtensionFiles: []string{filepath.Join(extensionPath, "index.ts"), filepath.Join(extensionPath, "security.ts")},
		ServerURL:      serverURL,
		Protocol:       "pi_extension",
		ConfigFormat:   "json",
		ConfigAction:   "add_extension_path",
		ConfigPatch:    piSettingsConfigPatch(extensionPath),
		GeneratedHookEvents: []string{
			string(goncho.HostHookPrompt),
			string(goncho.HostHookAssistantResponse),
			string(goncho.HostHookPreToolUse),
			string(goncho.HostHookPostToolUse),
			string(goncho.HostHookToolFailure),
			string(goncho.HostHookCompaction),
			string(goncho.HostHookSessionEnd),
		},
		MCPCommand: []string{"goncho-server", "serve", "-addr", serverAddr},
		RecommendedNextStep: []string{
			"Review the settings patch and TypeScript extension paths before copying files.",
			"Start goncho-server with the planned address before launching Pi.",
			"Keep --apply disabled until Pi extension files have smoke coverage with pi -e.",
		},
	}, nil
}

func piSettingsConfigPatch(extensionPath string) string {
	patch := map[string]any{
		"extensions": []string{extensionPath},
	}
	raw, err := json.MarshalIndent(patch, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(raw) + "\n"
}

func piRemoveSettingsConfigPatch(extensionPath string) string {
	patch := map[string]any{
		"remove_extensions": []string{extensionPath},
	}
	raw, err := json.MarshalIndent(patch, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(raw) + "\n"
}

func buildCodexConnectPlan(cfg config) (connectPlan, error) {
	configPath := strings.TrimSpace(cfg.ConfigPath)
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil || strings.TrimSpace(home) == "" {
			return connectPlan{}, errors.New("goncho connect codex: --config is required when home directory cannot be resolved")
		}
		configPath = filepath.Join(home, ".codex", "config.toml")
	}
	serverAddr := strings.TrimSpace(cfg.ServerAddr)
	if serverAddr == "" {
		serverAddr = "127.0.0.1:8765"
	}
	return connectPlan{
		Status:       "dry_run",
		Integration:  "codex",
		Mutates:      false,
		ConfigPath:   configPath,
		Protocol:     "mcp",
		ConfigFormat: "toml",
		ConfigAction: "append_or_replace_mcp_server",
		ConfigPatch:  codexMCPConfigPatch(serverAddr),
		GeneratedHookEvents: []string{
			string(goncho.HostHookPrompt),
			string(goncho.HostHookAssistantResponse),
			string(goncho.HostHookPreToolUse),
			string(goncho.HostHookPostToolUse),
			string(goncho.HostHookToolFailure),
			string(goncho.HostHookSessionEnd),
		},
		MCPCommand: []string{"goncho-server", "serve", "-addr", serverAddr},
		RecommendedNextStep: []string{
			"Review the TOML patch before applying it to Codex config.",
			"Start goncho-server with the planned address before launching Codex.",
			"Keep --apply disabled until Codex config golden tests and host smoke coverage are in place.",
		},
	}, nil
}

func codexMCPConfigPatch(serverAddr string) string {
	return fmt.Sprintf(`[mcp_servers.goncho]
command = "goncho-server"
args = ["serve", "-addr", %q]

[mcp_servers.goncho.env]
GONCHO_TRANSPORT = "http"
`, serverAddr)
}

func hostHookEventNames() []string {
	schemas := goncho.HostHookEventSchemas()
	out := make([]string, 0, len(schemas))
	for _, schema := range schemas {
		out = append(out, string(schema.Event))
	}
	return out
}

func validateGormesProfileID(profileID string) error {
	if strings.ContainsAny(profileID, `/\\`) || profileID == "." || profileID == ".." || strings.Contains(profileID, "..") {
		return fmt.Errorf("goncho connect gormes: unsafe profile %q", profileID)
	}
	return nil
}

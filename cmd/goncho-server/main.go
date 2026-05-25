package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	gonchohttp "github.com/TrebuchetDynamics/goncho/http"
	"github.com/TrebuchetDynamics/goncho/memory"
	goncho "github.com/TrebuchetDynamics/goncho/service"
)

const defaultServeAddr = "127.0.0.1:8765"

type config struct {
	Command        string
	DatabasePath   string
	ConfigPath     string
	Addr           string
	WorkspaceID    string
	ObserverPeerID string
	ImageDir       string
	VectorDir      string
	MaxDBBytes     int64
	MaxImageBytes  int64
	MaxVectorBytes int64
	AuthToken      string
	Stdin          io.Reader
	Stdout         io.Writer
	Stderr         io.Writer
}

type runtimeState struct {
	Store         *memory.SqliteStore
	DB            *sql.DB
	Service       *goncho.Service
	Tools         map[string]mcpTool
	DatabasePath  string
	WorkspaceID   string
	MigrationTime time.Time
}

type healthReport struct {
	Status     string                           `json:"status"`
	Version    string                           `json:"version"`
	DB         dbHealth                         `json:"db"`
	Migrations migrationHealth                  `json:"migrations"`
	Tools      toolHealth                       `json:"tools"`
	Providers  goncho.ProviderHealthDiagnostics `json:"providers"`
	Disk       goncho.DiskUsageDiagnostics      `json:"disk"`
}

type serverConfigFile struct {
	DatabasePath   string `json:"database_path"`
	WorkspaceID    string `json:"workspace_id"`
	ObserverPeerID string `json:"observer_peer_id"`
	ServeAddr      string `json:"serve_addr"`
	AuthMode       string `json:"auth_mode"`
	CreatedAt      string `json:"created_at"`
}

type initReport struct {
	Status     string `json:"status"`
	ConfigPath string `json:"config_path"`
	DBPath     string `json:"db_path"`
}

type onboardingReport struct {
	Status       string   `json:"status"`
	Mutates      bool     `json:"mutates"`
	DBPath       string   `json:"db_path"`
	ConfigPath   string   `json:"config_path"`
	BindAddr     string   `json:"bind_addr"`
	MCPURL       string   `json:"mcp_url"`
	NextCommands []string `json:"next_commands"`
	MCPSnippet   string   `json:"mcp_snippet"`
	HookSnippet  string   `json:"hook_snippet"`
}

type doctorReport struct {
	Status string        `json:"status"`
	DBPath string        `json:"db_path"`
	Addr   string        `json:"addr"`
	Checks []doctorCheck `json:"checks"`
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

type demoReport struct {
	Status     string    `json:"status"`
	DBPath     string    `json:"db_path"`
	Workspace  string    `json:"workspace_id"`
	Peer       string    `json:"peer_id"`
	SessionKey string    `json:"session_key"`
	Memory     string    `json:"memory"`
	Recall     demoProof `json:"recall"`
	Context    demoProof `json:"context"`
}

type demoProof struct {
	Proved        bool `json:"proved"`
	SelectedCount int  `json:"selected_count,omitempty"`
}

type dbHealth struct {
	Status string `json:"status"`
	Path   string `json:"path"`
}

type migrationHealth struct {
	Status    string `json:"status"`
	AppliedAt string `json:"applied_at,omitempty"`
}

type toolHealth struct {
	Count     int      `json:"count"`
	Available []string `json:"available"`
}

type mcpTool interface {
	Name() string
	Description() string
	Schema() json.RawMessage
	Timeout() time.Duration
	Execute(ctx context.Context, args json.RawMessage) (json.RawMessage, error)
}

type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *mcpError       `json:"error,omitempty"`
}

type mcpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type mcpToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type mcpResourceReadParams struct {
	URI        string `json:"uri"`
	ProfileID  string `json:"profile_id,omitempty"`
	Peer       string `json:"peer,omitempty"`
	Query      string `json:"query,omitempty"`
	SessionKey string `json:"session_key,omitempty"`
	Scope      string `json:"scope,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

type mcpPromptGetParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

var publicToolNames = []string{
	"goncho_context",
	"goncho_search",
	"goncho_recall",
	"goncho_remember",
	"goncho_review",
	"goncho_handoff",
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

func parseArgs(args []string) config {
	cfg, _ := parseConfig(args, io.Discard, io.Discard)
	return cfg
}

func parseConfig(args []string, stdout, stderr io.Writer) (config, error) {
	cfg := config{
		Command:      "serve",
		DatabasePath: defaultDBPath(),
		Addr:         defaultServeAddr,
		Stdin:        os.Stdin,
		Stdout:       stdout,
		Stderr:       stderr,
	}
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cfg.Command = strings.TrimSpace(args[0])
		args = args[1:]
	}
	fs := flag.NewFlagSet("goncho-server "+cfg.Command, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&cfg.DatabasePath, "db", cfg.DatabasePath, "SQLite database path")
	if cfg.Command == "init" || cfg.Command == "onboarding" {
		cfg.ConfigPath = defaultConfigPath()
		fs.StringVar(&cfg.ConfigPath, "config", cfg.ConfigPath, "goncho-server JSON config path")
	}
	fs.StringVar(&cfg.WorkspaceID, "workspace", "", "Goncho workspace ID; defaults to service default")
	fs.StringVar(&cfg.ObserverPeerID, "observer", "", "observer peer ID; defaults to service default")
	fs.StringVar(&cfg.ImageDir, "image-dir", "", "optional image storage directory for disk diagnostics")
	fs.StringVar(&cfg.VectorDir, "vector-dir", "", "optional vector index directory for disk diagnostics")
	fs.Int64Var(&cfg.MaxDBBytes, "max-db-bytes", 0, "optional DB size budget for diagnostics")
	fs.Int64Var(&cfg.MaxImageBytes, "max-image-bytes", 0, "optional image directory size budget for diagnostics")
	fs.Int64Var(&cfg.MaxVectorBytes, "max-vector-bytes", 0, "optional vector directory size budget for diagnostics")
	if cfg.Command == "serve" || cfg.Command == "doctor" || cfg.Command == "onboarding" {
		fs.StringVar(&cfg.Addr, "addr", cfg.Addr, "HTTP listen address to check or serve; defaults to loopback only")
	}
	if cfg.Command == "serve" {
		fs.StringVar(&cfg.AuthToken, "auth-token", "", "explicit local server token required for non-loopback binds; enforcement is reserved for future server mode")
	}
	if err := fs.Parse(args); err != nil {
		return cfg, err
	}
	if cfg.Command != "init" && cfg.Command != "onboarding" && cfg.Command != "serve" && cfg.Command != "stdio" && cfg.Command != "health" && cfg.Command != "demo" && cfg.Command != "doctor" && cfg.Command != "security" {
		return cfg, fmt.Errorf("goncho-server: unknown command %q (want init, onboarding, serve, stdio, health, demo, doctor, or security)", cfg.Command)
	}
	return cfg, nil
}

func run(ctx context.Context, cfg config) error {
	if cfg.Stdin == nil {
		cfg.Stdin = os.Stdin
	}
	if cfg.Stdout == nil {
		cfg.Stdout = os.Stdout
	}
	if cfg.Stderr == nil {
		cfg.Stderr = os.Stderr
	}
	switch cfg.Command {
	case "init":
		return runInit(ctx, cfg)
	case "onboarding":
		return runOnboarding(ctx, cfg)
	case "doctor":
		return runDoctor(ctx, cfg)
	case "security":
		return json.NewEncoder(cfg.Stdout).Encode(goncho.ServerModeSecurityRequirements())
	case "health":
		rt, err := openRuntime(ctx, cfg)
		if err != nil {
			return err
		}
		defer rt.Close(ctx)
		return json.NewEncoder(cfg.Stdout).Encode(rt.Health(ctx, retentionPolicyFromConfig(cfg)))
	case "demo":
		rt, err := openRuntime(ctx, cfg)
		if err != nil {
			return err
		}
		defer rt.Close(ctx)
		report := rt.RunDemo(ctx)
		if err := json.NewEncoder(cfg.Stdout).Encode(report); err != nil {
			return err
		}
		if report.Status != "ok" {
			return errors.New("goncho-server demo: recall/context proof failed")
		}
		return nil
	case "stdio":
		rt, err := openRuntime(ctx, cfg)
		if err != nil {
			return err
		}
		defer rt.Close(ctx)
		return runStdioMCP(ctx, cfg.Stdin, cfg.Stdout, rt)
	case "serve":
		if err := validateServeSecurity(cfg); err != nil {
			return err
		}
		rt, err := openRuntime(ctx, cfg)
		if err != nil {
			return err
		}
		defer rt.Close(ctx)
		server := &http.Server{
			Addr:              cfg.Addr,
			Handler:           newServerHandler(rt),
			ReadHeaderTimeout: 5 * time.Second,
		}
		fmt.Fprintf(cfg.Stderr, "goncho-server listening on http://%s\n", cfg.Addr)
		err = server.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	default:
		return fmt.Errorf("goncho-server: unknown command %q", cfg.Command)
	}
}

func retentionPolicyFromConfig(cfg config) goncho.RetentionPolicy {
	return goncho.RetentionPolicy{ImageDir: cfg.ImageDir, VectorDir: cfg.VectorDir, MaxDBBytes: cfg.MaxDBBytes, MaxImageBytes: cfg.MaxImageBytes, MaxVectorBytes: cfg.MaxVectorBytes}
}

func validateServeSecurity(cfg config) error {
	addr := strings.TrimSpace(cfg.Addr)
	if addr == "" {
		addr = defaultServeAddr
	}
	if isLoopbackListenAddr(addr) {
		return nil
	}
	if strings.TrimSpace(cfg.AuthToken) != "" {
		return nil
	}
	return fmt.Errorf("goncho-server serve: refusing unauthenticated non-loopback bind %q; use a loopback address like %s or configure an explicit server auth token", addr, defaultServeAddr)
}

func isLoopbackListenAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	host = strings.Trim(strings.TrimSpace(host), "[]")
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func runOnboarding(ctx context.Context, cfg config) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	dbPath := strings.TrimSpace(cfg.DatabasePath)
	if dbPath == "" {
		dbPath = defaultDBPath()
	}
	configPath := strings.TrimSpace(cfg.ConfigPath)
	if configPath == "" {
		configPath = defaultConfigPath()
	}
	addr := strings.TrimSpace(cfg.Addr)
	if addr == "" {
		addr = defaultServeAddr
	}
	report := onboardingReport{
		Status:     "plan",
		Mutates:    false,
		DBPath:     dbPath,
		ConfigPath: configPath,
		BindAddr:   addr,
		MCPURL:     "http://" + addr + "/mcp",
		NextCommands: []string{
			fmt.Sprintf("goncho-server init -db %q -config %q", dbPath, configPath),
			fmt.Sprintf("goncho-server serve -db %q -addr %q", dbPath, addr),
			"goncho connect <host> --plan to inspect host-specific MCP/hook changes before mutating files",
		},
		MCPSnippet:  fmt.Sprintf(`{"mcpServers":{"goncho":{"url":"http://%s/mcp"}}}`, addr),
		HookSnippet: "Forward approved host events to service.CaptureHostHook after applying local redaction policy; onboarding does not install hooks.",
	}
	return json.NewEncoder(cfg.Stdout).Encode(report)
}

func runDoctor(ctx context.Context, cfg config) error {
	dbPath := strings.TrimSpace(cfg.DatabasePath)
	if dbPath == "" {
		dbPath = defaultDBPath()
	}
	addr := strings.TrimSpace(cfg.Addr)
	if addr == "" {
		addr = defaultServeAddr
	}
	report := doctorReport{Status: "ok", DBPath: dbPath, Addr: addr}
	addCheck := func(name string, err error) {
		check := doctorCheck{Name: name, Status: "ok"}
		if err != nil {
			check.Status = "error"
			check.Message = err.Error()
			check.Suggestions = doctorSuggestions(name, dbPath, addr)
			report.Status = "error"
		}
		report.Checks = append(report.Checks, check)
	}

	addCheck("db_path", checkDBPath(dbPath))
	addCheck("write_permissions", checkWritePermissions(dbPath))
	rt, err := openRuntime(ctx, config{DatabasePath: dbPath, WorkspaceID: cfg.WorkspaceID, ObserverPeerID: cfg.ObserverPeerID})
	if err != nil {
		addCheck("migrations", err)
	} else {
		addCheck("migrations", nil)
	}
	addCheck("port_available", checkPortAvailable(addr))
	addCheck("public_tools", checkPublicTools())
	if rt != nil && rt.Service != nil {
		_, diskErr := rt.Service.DiskUsage(ctx, retentionPolicyFromConfig(cfg))
		addCheck("disk_usage", diskErr)
	} else {
		addCheck("disk_usage", err)
	}
	addDoctorProviderCheck := func(health goncho.ProviderHealthDiagnostics) {
		check := doctorCheck{Name: "optional_providers", Status: "ok"}
		for _, provider := range health {
			if provider.Status == goncho.ProviderStatusDegraded {
				check.Status = "degraded"
				check.Message = "one or more optional providers are degraded; core SQLite service remains usable"
				check.Suggestions = doctorSuggestions("optional_providers", dbPath, addr)
				break
			}
		}
		report.Checks = append(report.Checks, check)
	}
	addDoctorProviderCheck(rtForProviderHealth(rt))
	if rt != nil {
		_ = rt.Close(ctx)
	}

	return json.NewEncoder(cfg.Stdout).Encode(report)
}

func rtForProviderHealth(rt *runtimeState) goncho.ProviderHealthDiagnostics {
	if rt == nil || rt.Service == nil {
		return goncho.ProviderHealthDiagnostics{}
	}
	return rt.Service.ProviderHealthDiagnostics()
}

func doctorSuggestions(name, dbPath, addr string) []string {
	switch name {
	case "db_path", "write_permissions":
		return []string{
			fmt.Sprintf("mkdir -p %q", filepath.Dir(dbPath)),
			fmt.Sprintf("chmod 700 %q", filepath.Dir(dbPath)),
			fmt.Sprintf("goncho-server init -db %q", dbPath),
		}
	case "migrations":
		return []string{fmt.Sprintf("goncho-server init -db %q", dbPath), "Inspect the SQLite error above before retrying; doctor does not repair migrations automatically."}
	case "port_available":
		return []string{
			fmt.Sprintf("goncho-server serve -db %q -addr 127.0.0.1:0", dbPath),
			fmt.Sprintf("Choose a free loopback address instead of %q, then update connector plans with goncho connect <host> --plan -addr <addr>.", addr),
		}
	case "public_tools":
		return []string{"Run go test ./service ./cmd/goncho-server and inspect publicToolNames/buildMCPTools registration."}
	case "optional_providers":
		return []string{"Inspect provider health diagnostics; Goncho will continue lexical/graph recall fallback while optional providers recover.", "Increase provider timeout/cooldown only after confirming the adapter is local and trusted."}
	default:
		return nil
	}
}

func runInit(ctx context.Context, cfg config) error {
	rt, err := openRuntime(ctx, cfg)
	if err != nil {
		return err
	}
	defer rt.Close(ctx)
	configPath := strings.TrimSpace(cfg.ConfigPath)
	if configPath == "" {
		configPath = defaultConfigPath()
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		return fmt.Errorf("goncho-server init: create config dir: %w", err)
	}
	serverCfg := serverConfigFile{
		DatabasePath:   rt.DatabasePath,
		WorkspaceID:    rt.WorkspaceID,
		ObserverPeerID: effectiveObserverPeerID(cfg.ObserverPeerID),
		ServeAddr:      defaultServeAddr,
		AuthMode:       "loopback_only",
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
	}
	raw, err := json.MarshalIndent(serverCfg, "", "  ")
	if err != nil {
		return fmt.Errorf("goncho-server init: encode config: %w", err)
	}
	raw = append(raw, '\n')
	configFile, err := os.OpenFile(configPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("goncho-server init: create config: %w", err)
	}
	if _, err := configFile.Write(raw); err != nil {
		_ = configFile.Close()
		return fmt.Errorf("goncho-server init: write config: %w", err)
	}
	if err := configFile.Close(); err != nil {
		return fmt.Errorf("goncho-server init: close config: %w", err)
	}
	return json.NewEncoder(cfg.Stdout).Encode(initReport{Status: "ok", ConfigPath: configPath, DBPath: rt.DatabasePath})
}

func openRuntime(ctx context.Context, cfg config) (*runtimeState, error) {
	dbPath := strings.TrimSpace(cfg.DatabasePath)
	if dbPath == "" {
		dbPath = defaultDBPath()
	}
	store, err := memory.OpenSqlite(dbPath, 0, slog.Default())
	if err != nil {
		return nil, fmt.Errorf("goncho-server: open sqlite: %w", err)
	}
	if err := goncho.RunMigrations(store.DB()); err != nil {
		_ = store.Close(ctx)
		return nil, fmt.Errorf("goncho-server: run migrations: %w", err)
	}
	serviceCfg := goncho.Config{
		WorkspaceID:     cfg.WorkspaceID,
		ObserverPeerID:  cfg.ObserverPeerID,
		PeerCardEnabled: true,
	}.Effective()
	svc := goncho.NewService(store.DB(), serviceCfg, slog.Default())
	tools := buildMCPTools(svc, store.DB(), dbPath, serviceCfg)
	return &runtimeState{Store: store, DB: store.DB(), Service: svc, Tools: tools, DatabasePath: dbPath, WorkspaceID: serviceCfg.WorkspaceID, MigrationTime: time.Now().UTC()}, nil
}

func (rt *runtimeState) Close(ctx context.Context) error {
	if rt == nil || rt.Store == nil {
		return nil
	}
	return rt.Store.Close(ctx)
}

func (rt *runtimeState) RunDemo(ctx context.Context) demoReport {
	workspaceID := goncho.DefaultWorkspaceID
	if rt != nil && strings.TrimSpace(rt.WorkspaceID) != "" {
		workspaceID = rt.WorkspaceID
	}
	peerID := "demo-operator"
	sessionKey := "demo-project-memory"
	query := "quartz llama launch checklist owner"
	memory := "Project Quartz Llama launch checklist owner is Mira; verify recall before action."
	report := demoReport{Status: "ok", DBPath: rt.DatabasePath, Workspace: workspaceID, Peer: peerID, SessionKey: sessionKey, Memory: memory}
	if _, err := rt.Service.Conclude(ctx, goncho.ConcludeParams{Peer: peerID, SessionKey: sessionKey, Conclusion: memory, Scope: goncho.MemoryScopeWorkspace}); err != nil {
		report.Status = "error"
		return report
	}
	recall, err := rt.Service.Recall(ctx, goncho.RecallQuery{WorkspaceID: workspaceID, Peer: peerID, SessionKey: sessionKey, Query: query, Limit: 5})
	if err == nil {
		report.Recall.SelectedCount = len(recall.Selected)
		report.Recall.Proved = recallSelectedContains(recall.Selected, memory)
	}
	contextResult, err := rt.Service.Context(ctx, goncho.ContextParams{Peer: peerID, SessionKey: sessionKey, Query: query})
	if err == nil {
		report.Context.Proved = stringSliceContains(contextResult.Conclusions, memory)
	}
	if !report.Recall.Proved || !report.Context.Proved {
		report.Status = "error"
	}
	return report
}

func (rt *runtimeState) Health(ctx context.Context, policies ...goncho.RetentionPolicy) healthReport {
	policy := goncho.RetentionPolicy{}
	if len(policies) > 0 {
		policy = policies[0]
	}
	report := healthReport{
		Status:  "ok",
		Version: buildVersion(),
		DB: dbHealth{
			Status: "ok",
			Path:   rt.DatabasePath,
		},
		Migrations: migrationHealth{
			Status:    "ok",
			AppliedAt: rt.MigrationTime.Format(time.RFC3339),
		},
		Tools: toolHealth{
			Count:     len(publicToolNames),
			Available: append([]string(nil), publicToolNames...),
		},
		Providers: rt.Service.ProviderHealthDiagnostics(),
	}
	if rt != nil && rt.Service != nil {
		disk, err := rt.Service.DiskUsage(ctx, policy)
		if err == nil {
			report.Disk = disk
		}
	}
	if rt == nil || rt.DB == nil {
		report.Status = "error"
		report.DB.Status = "error"
		return report
	}
	if err := rt.DB.PingContext(ctx); err != nil {
		report.Status = "error"
		report.DB.Status = "error"
	}
	return report
}

func newServerHandler(rt *runtimeState) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, rt.Health(r.Context()))
	})
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleMCP(w, r, rt)
	})
	mux.Handle("/", gonchohttp.NewServiceHandler(rt.Service))
	return mux
}

func buildMCPTools(svc *goncho.Service, db *sql.DB, dbPath string, cfg goncho.Config) map[string]mcpTool {
	memoryStore := goncho.NewLocalMarkdownMemoryStore(db, goncho.LocalMarkdownMemoryConfig{
		Path:           filepath.Join(filepath.Dir(dbPath), "goncho-server-memory.md"),
		WorkspaceID:    cfg.WorkspaceID,
		ObserverPeerID: cfg.ObserverPeerID,
		PeerID:         "goncho-server",
		SessionID:      "goncho-server",
	})
	tools := []mcpTool{
		goncho.NewGonchoContextTool(svc),
		goncho.NewGonchoSearchTool(svc),
		goncho.NewGonchoRecallTool(svc),
		goncho.NewGonchoRememberTool(svc),
		goncho.NewReviewTool(svc),
		goncho.NewGonchoHandoffTool(memoryStore),
	}
	out := make(map[string]mcpTool, len(tools))
	for _, tool := range tools {
		out[tool.Name()] = tool
	}
	return out
}

func handleMCP(w http.ResponseWriter, r *http.Request, rt *runtimeState) {
	var req mcpRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusOK, mcpResponse{JSONRPC: "2.0", Error: &mcpError{Code: -32700, Message: err.Error()}})
		return
	}
	writeJSON(w, http.StatusOK, handleMCPRequest(r.Context(), rt, req))
}

func handleMCPRequest(ctx context.Context, rt *runtimeState, req mcpRequest) mcpResponse {
	resp := mcpResponse{JSONRPC: "2.0", ID: req.ID}
	if req.JSONRPC != "2.0" || strings.TrimSpace(req.Method) == "" {
		resp.Error = &mcpError{Code: -32600, Message: "invalid JSON-RPC request"}
		return resp
	}
	switch req.Method {
	case "initialize":
		resp.Result = map[string]any{"protocolVersion": "2024-11-05", "serverInfo": map[string]any{"name": "goncho-server", "version": buildVersion()}, "capabilities": map[string]any{"tools": map[string]any{}, "resources": map[string]any{}, "prompts": map[string]any{}}}
	case "tools/list":
		resp.Result = map[string]any{"tools": mcpToolDescriptors(rt.Tools)}
	case "resources/list":
		resp.Result = map[string]any{"resources": mcpResourceDescriptors(rt.Service)}
	case "resources/read":
		result, err := mcpReadResource(ctx, rt.Service, req.Params)
		if err != nil {
			resp.Error = &mcpError{Code: -32602, Message: err.Error()}
		} else {
			resp.Result = result
		}
	case "prompts/list":
		resp.Result = map[string]any{"prompts": mcpPromptDescriptors(rt.Service)}
	case "prompts/get":
		result, err := mcpGetPrompt(ctx, rt.Service, req.Params)
		if err != nil {
			resp.Error = &mcpError{Code: -32602, Message: err.Error()}
		} else {
			resp.Result = result
		}
	case "tools/call":
		result, err := mcpCallTool(ctx, rt.Tools, req.Params)
		if err != nil {
			resp.Error = &mcpError{Code: -32602, Message: err.Error()}
		} else {
			resp.Result = result
		}
	case "ping":
		resp.Result = map[string]any{}
	default:
		resp.Error = &mcpError{Code: -32601, Message: "method not found"}
	}
	return resp
}

func runStdioMCP(ctx context.Context, in io.Reader, out io.Writer, rt *runtimeState) error {
	scanner := bufio.NewScanner(in)
	encoder := json.NewEncoder(out)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var req mcpRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			if encodeErr := encoder.Encode(mcpResponse{JSONRPC: "2.0", Error: &mcpError{Code: -32700, Message: err.Error()}}); encodeErr != nil {
				return encodeErr
			}
			continue
		}
		if err := encoder.Encode(handleMCPRequest(ctx, rt, req)); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func mcpResourceDescriptors(svc *goncho.Service) []map[string]any {
	registry := goncho.NewMemoryResourceRegistry(svc)
	descriptors := registry.Descriptors()
	out := make([]map[string]any, 0, len(descriptors))
	for _, descriptor := range descriptors {
		if descriptor.Kind != goncho.MemoryResourceKindResource {
			continue
		}
		out = append(out, map[string]any{"uri": descriptor.URI, "name": descriptor.Name, "description": descriptor.Description, "mimeType": descriptor.MimeType})
	}
	return out
}

func mcpPromptDescriptors(svc *goncho.Service) []map[string]any {
	registry := goncho.NewMemoryResourceRegistry(svc)
	descriptors := registry.Descriptors()
	out := make([]map[string]any, 0, len(descriptors))
	for _, descriptor := range descriptors {
		if descriptor.Kind != goncho.MemoryResourceKindPrompt {
			continue
		}
		out = append(out, map[string]any{"name": descriptor.Name, "description": descriptor.Description, "arguments": []map[string]string{{"name": "peer", "description": "Goncho peer id"}, {"name": "session_key", "description": "Optional session key"}, {"name": "query", "description": "Optional recall or verification query"}}})
	}
	return out
}

func mcpToolDescriptors(tools map[string]mcpTool) []map[string]any {
	out := make([]map[string]any, 0, len(publicToolNames))
	for _, name := range publicToolNames {
		tool, ok := tools[name]
		if !ok {
			continue
		}
		out = append(out, map[string]any{"name": tool.Name(), "description": tool.Description(), "inputSchema": json.RawMessage(tool.Schema())})
	}
	return out
}

func mcpReadResource(ctx context.Context, svc *goncho.Service, rawParams json.RawMessage) (map[string]any, error) {
	var params mcpResourceReadParams
	if len(rawParams) == 0 {
		return nil, errors.New("resources/read requires params")
	}
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}
	content, err := goncho.NewMemoryResourceRegistry(svc).Read(ctx, goncho.MemoryResourceRequest{URI: params.URI, ProfileID: params.ProfileID, Peer: params.Peer, Query: params.Query, SessionKey: params.SessionKey, Scope: params.Scope, Limit: params.Limit})
	if err != nil {
		return nil, err
	}
	text, err := mcpResourceText(content)
	if err != nil {
		return nil, err
	}
	return map[string]any{"contents": []map[string]any{{"uri": content.URI, "mimeType": content.MimeType, "text": text}}}, nil
}

func mcpGetPrompt(ctx context.Context, svc *goncho.Service, rawParams json.RawMessage) (map[string]any, error) {
	var params mcpPromptGetParams
	if len(rawParams) == 0 {
		return nil, errors.New("prompts/get requires params")
	}
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}
	var args mcpResourceReadParams
	if len(params.Arguments) > 0 && string(params.Arguments) != "null" {
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			return nil, err
		}
	}
	uri, err := mcpPromptURI(params.Name)
	if err != nil {
		return nil, err
	}
	content, err := goncho.NewMemoryResourceRegistry(svc).Read(ctx, goncho.MemoryResourceRequest{URI: uri, ProfileID: args.ProfileID, Peer: args.Peer, Query: args.Query, SessionKey: args.SessionKey, Scope: args.Scope, Limit: args.Limit})
	if err != nil {
		return nil, err
	}
	text, err := mcpResourceText(content)
	if err != nil {
		return nil, err
	}
	return map[string]any{"description": "Goncho " + params.Name + " prompt", "messages": []map[string]any{{"role": "user", "content": map[string]string{"type": "text", "text": text}}}}, nil
}

func mcpPromptURI(name string) (string, error) {
	switch strings.TrimSpace(name) {
	case "recall_prompt":
		return "goncho://recall/prompt", nil
	case "session_handoff":
		return "goncho://handoff/prompt", nil
	case "review_resolution":
		return "goncho://review/prompt", nil
	case "verification_before_action":
		return "goncho://verify/prompt", nil
	default:
		return "", fmt.Errorf("unknown prompt %q", name)
	}
}

func mcpResourceText(content goncho.MemoryResourceContent) (string, error) {
	if content.MimeType == "text/plain" {
		text, _ := content.Payload["prompt"].(string)
		return text, nil
	}
	raw, err := json.Marshal(content.Payload)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func mcpCallTool(ctx context.Context, tools map[string]mcpTool, rawParams json.RawMessage) (map[string]any, error) {
	var params mcpToolCallParams
	if len(rawParams) == 0 {
		return nil, errors.New("tools/call requires params")
	}
	if err := json.Unmarshal(rawParams, &params); err != nil {
		return nil, err
	}
	tool, ok := tools[params.Name]
	if !ok {
		return nil, fmt.Errorf("unknown tool %q", params.Name)
	}
	args := params.Arguments
	if len(args) == 0 || string(args) == "null" {
		args = json.RawMessage(`{}`)
	}
	if timeout := tool.Timeout(); timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	out, err := tool.Execute(ctx, args)
	if err != nil {
		return nil, err
	}
	return map[string]any{"content": []map[string]string{{"type": "text", "text": string(out)}}}, nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func checkDBPath(dbPath string) error {
	if strings.TrimSpace(dbPath) == "" {
		return errors.New("database path is required")
	}
	if info, err := os.Stat(dbPath); err == nil && info.IsDir() {
		return fmt.Errorf("database path %q is a directory", dbPath)
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.MkdirAll(filepath.Dir(dbPath), 0o700)
}

func checkWritePermissions(dbPath string) error {
	if err := checkDBPath(dbPath); err != nil {
		return err
	}
	probe, err := os.CreateTemp(filepath.Dir(dbPath), ".goncho-write-probe-*")
	if err != nil {
		return err
	}
	probePath := probe.Name()
	if _, err := probe.Write([]byte("ok")); err != nil {
		_ = probe.Close()
		_ = os.Remove(probePath)
		return err
	}
	if err := probe.Close(); err != nil {
		_ = os.Remove(probePath)
		return err
	}
	return os.Remove(probePath)
}

func checkPortAvailable(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return listener.Close()
}

func checkPublicTools() error {
	if len(publicToolNames) == 0 {
		return errors.New("no public tools registered")
	}
	seen := map[string]bool{}
	for _, name := range publicToolNames {
		name = strings.TrimSpace(name)
		if name == "" {
			return errors.New("blank public tool name")
		}
		if seen[name] {
			return fmt.Errorf("duplicate public tool %q", name)
		}
		seen[name] = true
	}
	return nil
}

func recallSelectedContains(selected []goncho.ScoredRecallCandidate, content string) bool {
	for _, item := range selected {
		if item.Candidate.Content == content {
			return true
		}
	}
	return false
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || strings.TrimSpace(info.Main.Version) == "" {
		return "(devel)"
	}
	return info.Main.Version
}

func defaultDBPath() string {
	base, err := os.UserConfigDir()
	if err != nil || strings.TrimSpace(base) == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "goncho", "goncho.db")
}

func defaultConfigPath() string {
	base, err := os.UserConfigDir()
	if err != nil || strings.TrimSpace(base) == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "goncho", "goncho-server.json")
}

func effectiveObserverPeerID(observer string) string {
	observer = strings.TrimSpace(observer)
	if observer == "" {
		return goncho.DefaultObserverPeerID
	}
	return observer
}

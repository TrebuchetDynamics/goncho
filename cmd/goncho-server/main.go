package main

import (
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
	Status     string          `json:"status"`
	Version    string          `json:"version"`
	DB         dbHealth        `json:"db"`
	Migrations migrationHealth `json:"migrations"`
	Tools      toolHealth      `json:"tools"`
}

type serverConfigFile struct {
	DatabasePath   string `json:"database_path"`
	WorkspaceID    string `json:"workspace_id"`
	ObserverPeerID string `json:"observer_peer_id"`
	ServeAddr      string `json:"serve_addr"`
	CreatedAt      string `json:"created_at"`
}

type initReport struct {
	Status     string `json:"status"`
	ConfigPath string `json:"config_path"`
	DBPath     string `json:"db_path"`
}

type doctorReport struct {
	Status string        `json:"status"`
	DBPath string        `json:"db_path"`
	Addr   string        `json:"addr"`
	Checks []doctorCheck `json:"checks"`
}

type doctorCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
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
	if cfg.Command == "init" {
		cfg.ConfigPath = defaultConfigPath()
		fs.StringVar(&cfg.ConfigPath, "config", cfg.ConfigPath, "goncho-server JSON config path")
	}
	fs.StringVar(&cfg.WorkspaceID, "workspace", "", "Goncho workspace ID; defaults to service default")
	fs.StringVar(&cfg.ObserverPeerID, "observer", "", "observer peer ID; defaults to service default")
	if cfg.Command == "serve" || cfg.Command == "doctor" {
		fs.StringVar(&cfg.Addr, "addr", cfg.Addr, "HTTP listen address to check or serve; defaults to loopback only")
	}
	if err := fs.Parse(args); err != nil {
		return cfg, err
	}
	if cfg.Command != "init" && cfg.Command != "serve" && cfg.Command != "health" && cfg.Command != "demo" && cfg.Command != "doctor" {
		return cfg, fmt.Errorf("goncho-server: unknown command %q (want init, serve, health, demo, or doctor)", cfg.Command)
	}
	return cfg, nil
}

func run(ctx context.Context, cfg config) error {
	if cfg.Stdout == nil {
		cfg.Stdout = os.Stdout
	}
	if cfg.Stderr == nil {
		cfg.Stderr = os.Stderr
	}
	switch cfg.Command {
	case "init":
		return runInit(ctx, cfg)
	case "doctor":
		return runDoctor(ctx, cfg)
	case "health":
		rt, err := openRuntime(ctx, cfg)
		if err != nil {
			return err
		}
		defer rt.Close(ctx)
		return json.NewEncoder(cfg.Stdout).Encode(rt.Health(ctx))
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
	case "serve":
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
		_ = rt.Close(ctx)
	}
	addCheck("port_available", checkPortAvailable(addr))
	addCheck("public_tools", checkPublicTools())

	return json.NewEncoder(cfg.Stdout).Encode(report)
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

func (rt *runtimeState) Health(ctx context.Context) healthReport {
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
	resp := mcpResponse{JSONRPC: "2.0", ID: req.ID}
	switch req.Method {
	case "initialize":
		resp.Result = map[string]any{"protocolVersion": "2024-11-05", "serverInfo": map[string]any{"name": "goncho-server", "version": buildVersion()}, "capabilities": map[string]any{"tools": map[string]any{}}}
	case "tools/list":
		resp.Result = map[string]any{"tools": mcpToolDescriptors(rt.Tools)}
	case "tools/call":
		result, err := mcpCallTool(r.Context(), rt.Tools, req.Params)
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
	writeJSON(w, http.StatusOK, resp)
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

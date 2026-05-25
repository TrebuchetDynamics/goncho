package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestRunInitCreatesConfigAndSQLiteDB(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "goncho.db")
	configPath := filepath.Join(dir, "goncho-server.json")
	var stdout bytes.Buffer

	if err := run(context.Background(), config{Command: "init", DatabasePath: dbPath, ConfigPath: configPath, WorkspaceID: "init-workspace", ObserverPeerID: "init-observer", Stdout: &stdout}); err != nil {
		t.Fatalf("run init: %v", err)
	}
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("stat db path: %v", err)
	}
	rawConfig, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var cfg serverConfigFile
	if err := json.Unmarshal(rawConfig, &cfg); err != nil {
		t.Fatalf("decode config: %v\n%s", err, rawConfig)
	}
	if cfg.DatabasePath != dbPath || cfg.WorkspaceID != "init-workspace" || cfg.ObserverPeerID != "init-observer" || cfg.ServeAddr != defaultServeAddr {
		t.Fatalf("config = %+v, want db/workspace/observer/default addr", cfg)
	}
	var report initReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode init report: %v\n%s", err, stdout.String())
	}
	if report.Status != "ok" || report.ConfigPath != configPath || report.DBPath != dbPath {
		t.Fatalf("init report = %+v, want ok paths", report)
	}
}

func TestRunDoctorReportsDBMigrationsWritePortAndTools(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "goncho.db")
	addr := freeTestAddr(t)
	var stdout bytes.Buffer

	if err := run(context.Background(), config{Command: "doctor", DatabasePath: dbPath, Addr: addr, Stdout: &stdout}); err != nil {
		t.Fatalf("run doctor: %v", err)
	}
	var report doctorReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode doctor report: %v\n%s", err, stdout.String())
	}
	if report.Status != "ok" || report.DBPath != dbPath || report.Addr != addr {
		t.Fatalf("doctor report = %+v, want ok/db/addr", report)
	}
	for _, name := range []string{"db_path", "migrations", "write_permissions", "port_available", "public_tools"} {
		check, ok := report.CheckByName(name)
		if !ok {
			t.Fatalf("doctor missing check %q in %+v", name, report.Checks)
		}
		if check.Status != "ok" {
			t.Fatalf("doctor check %q = %+v, want ok", name, check)
		}
	}
}

func TestRunDoctorReportsPortConflict(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()
	var stdout bytes.Buffer

	if err := run(context.Background(), config{Command: "doctor", DatabasePath: filepath.Join(t.TempDir(), "goncho.db"), Addr: listener.Addr().String(), Stdout: &stdout}); err != nil {
		t.Fatalf("run doctor: %v", err)
	}
	var report doctorReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode doctor report: %v\n%s", err, stdout.String())
	}
	check, ok := report.CheckByName("port_available")
	if report.Status != "error" || !ok || check.Status != "error" {
		t.Fatalf("doctor port conflict report = %+v check=%+v ok=%v, want error", report, check, ok)
	}
}

func TestRunHealthReportsDBMigrationsAndTools(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "goncho.db")
	var stdout bytes.Buffer

	if err := run(context.Background(), config{Command: "health", DatabasePath: dbPath, Stdout: &stdout}); err != nil {
		t.Fatalf("run health: %v", err)
	}

	var report healthReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode health report: %v\n%s", err, stdout.String())
	}
	if report.Status != "ok" {
		t.Fatalf("status = %q, want ok: %+v", report.Status, report)
	}
	if report.DB.Path != dbPath || report.DB.Status != "ok" {
		t.Fatalf("db health = %+v, want ok path %q", report.DB, dbPath)
	}
	if report.Migrations.Status != "ok" {
		t.Fatalf("migration health = %+v, want ok", report.Migrations)
	}
	wantTools := []string{"goncho_context", "goncho_search", "goncho_recall", "goncho_remember", "goncho_review", "goncho_handoff"}
	if !slices.Equal(report.Tools.Available, wantTools) || report.Tools.Count != len(wantTools) {
		t.Fatalf("tools = %+v, want %v", report.Tools, wantTools)
	}
	if report.Version == "" {
		t.Fatalf("version is empty in %+v", report)
	}
}

func TestRunDemoSeedsAndProvesRecallContext(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "goncho.db")
	var stdout bytes.Buffer

	if err := run(context.Background(), config{Command: "demo", DatabasePath: dbPath, Stdout: &stdout}); err != nil {
		t.Fatalf("run demo: %v", err)
	}

	var report demoReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode demo report: %v\n%s", err, stdout.String())
	}
	if report.Status != "ok" || report.DBPath != dbPath {
		t.Fatalf("demo report = %+v, want ok with db path %q", report, dbPath)
	}
	if !report.Recall.Proved || report.Recall.SelectedCount == 0 {
		t.Fatalf("demo recall proof = %+v, want proved selected recall", report.Recall)
	}
	if !report.Context.Proved {
		t.Fatalf("demo context proof = %+v, want proved context", report.Context)
	}
}

func TestServerHandlerExposesMCPToolsListAndCall(t *testing.T) {
	rt, err := openRuntime(context.Background(), config{DatabasePath: filepath.Join(t.TempDir(), "goncho.db")})
	if err != nil {
		t.Fatalf("open runtime: %v", err)
	}
	defer rt.Close(context.Background())
	handler := newServerHandler(rt)

	listed := postMCP(t, handler, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list"})
	tools := listed["result"].(map[string]any)["tools"].([]any)
	if !mcpToolsContain(tools, "goncho_remember") || !mcpToolsContain(tools, "goncho_search") {
		t.Fatalf("tools/list = %#v, want goncho_remember and goncho_search", tools)
	}

	remembered := postMCP(t, handler, map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": map[string]any{"name": "goncho_remember", "arguments": map[string]any{"peer_id": "mcp-peer", "content": "MCP transport remembers quartz mule.", "session_key": "mcp-session"}}})
	if remembered["error"] != nil {
		t.Fatalf("remember call returned error: %#v", remembered)
	}
	searched := postMCP(t, handler, map[string]any{"jsonrpc": "2.0", "id": 3, "method": "tools/call", "params": map[string]any{"name": "goncho_search", "arguments": map[string]any{"peer_id": "mcp-peer", "query": "quartz mule", "session_key": "mcp-session"}}})
	if !strings.Contains(mcpTextContent(searched), "MCP transport remembers quartz mule") {
		t.Fatalf("search call = %#v, want remembered content", searched)
	}
}

func TestServerHandlerExposesHealthAndLocalHTTPAdapter(t *testing.T) {
	rt, err := openRuntime(context.Background(), config{DatabasePath: filepath.Join(t.TempDir(), "goncho.db")})
	if err != nil {
		t.Fatalf("open runtime: %v", err)
	}
	defer rt.Close(context.Background())

	handler := newServerHandler(rt)

	healthReq := httptest.NewRequest(http.MethodGet, "/health", nil)
	healthRec := httptest.NewRecorder()
	handler.ServeHTTP(healthRec, healthReq)
	if healthRec.Code != http.StatusOK {
		t.Fatalf("/health status = %d body=%s", healthRec.Code, healthRec.Body.String())
	}

	missingReq := httptest.NewRequest(http.MethodGet, "/v3/workspaces/default/peers/alice/context", nil)
	missingRec := httptest.NewRecorder()
	handler.ServeHTTP(missingRec, missingReq)
	if missingRec.Code != http.StatusOK {
		t.Fatalf("service route status = %d body=%s", missingRec.Code, missingRec.Body.String())
	}
}

func TestDefaultServeAddressIsLoopbackOnly(t *testing.T) {
	cfg := parseArgs([]string{"serve", "-db", filepath.Join(t.TempDir(), "goncho.db")})
	if cfg.Addr != defaultServeAddr {
		t.Fatalf("addr = %q, want %q", cfg.Addr, defaultServeAddr)
	}
}

func postMCP(t *testing.T, handler http.Handler, body map[string]any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal MCP request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("MCP status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode MCP response: %v\n%s", err, rec.Body.String())
	}
	return out
}

func mcpToolsContain(tools []any, name string) bool {
	for _, tool := range tools {
		obj, ok := tool.(map[string]any)
		if ok && obj["name"] == name {
			return true
		}
	}
	return false
}

func mcpTextContent(response map[string]any) string {
	result, _ := response["result"].(map[string]any)
	content, _ := result["content"].([]any)
	var out string
	for _, item := range content {
		obj, _ := item.(map[string]any)
		text, _ := obj["text"].(string)
		out += text
	}
	return out
}

func freeTestAddr(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for free addr: %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close free addr listener: %v", err)
	}
	return addr
}

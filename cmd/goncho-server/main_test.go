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
	"time"

	goncho "github.com/TrebuchetDynamics/goncho/service"
)

func TestRunOnboardingPrintsNonMutatingGuidance(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "goncho.db")
	configPath := filepath.Join(dir, "goncho-server.json")
	var stdout bytes.Buffer

	if err := run(context.Background(), config{Command: "onboarding", DatabasePath: dbPath, ConfigPath: configPath, Addr: "127.0.0.1:8799", Stdout: &stdout}); err != nil {
		t.Fatalf("run onboarding: %v", err)
	}
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Fatalf("onboarding created db path or unexpected stat error: %v", err)
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("onboarding created config path or unexpected stat error: %v", err)
	}
	var report onboardingReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode onboarding report: %v\n%s", err, stdout.String())
	}
	if report.Status != "plan" || report.Mutates || report.DBPath != dbPath || report.ConfigPath != configPath || report.BindAddr != "127.0.0.1:8799" || report.MCPURL != "http://127.0.0.1:8799/mcp" {
		t.Fatalf("onboarding report = %+v, want non-mutating local paths and MCP URL", report)
	}
	for _, want := range []string{"goncho-server init", "goncho-server serve", "goncho connect"} {
		if !strings.Contains(strings.Join(report.NextCommands, "\n"), want) {
			t.Fatalf("next commands = %v, missing %q", report.NextCommands, want)
		}
	}
	if !strings.Contains(report.MCPSnippet, "127.0.0.1:8799") || !strings.Contains(report.HookSnippet, "CaptureHostHook") {
		t.Fatalf("snippets = %q / %q, want MCP addr and hook guidance", report.MCPSnippet, report.HookSnippet)
	}
}

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
	for _, name := range []string{"db_path", "migrations", "write_permissions", "port_available", "public_tools", "disk_usage", "optional_providers"} {
		check, ok := report.CheckByName(name)
		if !ok {
			t.Fatalf("doctor missing check %q in %+v", name, report.Checks)
		}
		if check.Status != "ok" {
			t.Fatalf("doctor check %q = %+v, want ok", name, check)
		}
	}
}

func TestRunDoctorReportsAutofixSuggestionsWithoutApplyingThem(t *testing.T) {
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
	if !ok || len(check.Suggestions) == 0 {
		t.Fatalf("port check = %+v ok=%v, want autofix suggestions", check, ok)
	}
	if !strings.Contains(strings.Join(check.Suggestions, "\n"), "goncho-server serve") || !strings.Contains(strings.Join(check.Suggestions, "\n"), "-addr") {
		t.Fatalf("suggestions = %v, want copy-paste serve command with alternate addr", check.Suggestions)
	}
	if _, err := net.Listen("tcp", listener.Addr().String()); err == nil {
		t.Fatal("doctor unexpectedly freed or changed the occupied port")
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
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "goncho.db")
	imageDir := filepath.Join(dir, "images")
	vectorDir := filepath.Join(dir, "vectors")
	if err := os.MkdirAll(imageDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(vectorDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(imageDir, "img.bin"), []byte("image"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vectorDir, "vec.bin"), []byte("vector"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer

	if err := run(context.Background(), config{Command: "health", DatabasePath: dbPath, ImageDir: imageDir, VectorDir: vectorDir, MaxDBBytes: 1, MaxImageBytes: 1, MaxVectorBytes: 1, Stdout: &stdout}); err != nil {
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
	if report.Disk.DB.Bytes == 0 || report.Disk.Images.Bytes == 0 || report.Disk.Vectors.Bytes == 0 || !report.Disk.DB.OverBudget || !report.Disk.Images.OverBudget || !report.Disk.Vectors.OverBudget {
		t.Fatalf("disk = %+v, want DB/image/vector over-budget diagnostics", report.Disk)
	}
	if got := report.Providers.ByName("embedding"); got.Name != "embedding" || got.Status == "" {
		t.Fatalf("provider health = %+v, want embedding optional provider diagnostics", report.Providers)
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

func TestMCPToolCallHonorsCancellationAndTimeouts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := mcpCallTool(ctx, map[string]mcpTool{"slow": slowMCPTool{timeout: time.Second}}, json.RawMessage(`{"name":"slow","arguments":{}}`))
	if err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("canceled mcpCallTool err = %v, want context canceled", err)
	}

	_, err = mcpCallTool(context.Background(), map[string]mcpTool{"slow": slowMCPTool{timeout: time.Nanosecond}}, json.RawMessage(`{"name":"slow","arguments":{}}`))
	if err == nil || !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("timed out mcpCallTool err = %v, want deadline exceeded", err)
	}
}

func TestRunStdioMCPHandlesInitializePingAndInvalidRequests(t *testing.T) {
	var stdout bytes.Buffer
	stdin := strings.NewReader(strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize"}`,
		`{"jsonrpc":"2.0","id":2,"method":"ping"}`,
		`{"jsonrpc":"1.0","id":3,"method":"ping"}`,
	}, "\n"))

	if err := run(context.Background(), config{Command: "stdio", DatabasePath: filepath.Join(t.TempDir(), "goncho.db"), Stdin: stdin, Stdout: &stdout}); err != nil {
		t.Fatalf("run stdio: %v", err)
	}
	responses := decodeMCPResponses(t, stdout.String())
	if len(responses) != 3 {
		t.Fatalf("responses = %#v, want 3", responses)
	}
	if responses[0]["error"] != nil || responses[0]["result"].(map[string]any)["capabilities"] == nil {
		t.Fatalf("initialize stdio response = %#v", responses[0])
	}
	if responses[1]["error"] != nil {
		t.Fatalf("ping stdio response = %#v", responses[1])
	}
	if errObj, ok := responses[2]["error"].(map[string]any); !ok || int(errObj["code"].(float64)) != -32600 {
		t.Fatalf("invalid stdio response = %#v, want -32600", responses[2])
	}
}

func TestServerHandlerExposesMCPResourcesAndPrompts(t *testing.T) {
	rt, err := openRuntime(context.Background(), config{DatabasePath: filepath.Join(t.TempDir(), "goncho.db")})
	if err != nil {
		t.Fatalf("open runtime: %v", err)
	}
	defer rt.Close(context.Background())
	if _, err := rt.Service.Conclude(context.Background(), goncho.ConcludeParams{Peer: "mcp-peer", SessionKey: "mcp-session", Conclusion: "MCP resources expose local evidence."}); err != nil {
		t.Fatalf("seed conclusion: %v", err)
	}
	handler := newServerHandler(rt)

	initialized := postMCP(t, handler, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize"})
	caps := initialized["result"].(map[string]any)["capabilities"].(map[string]any)
	if caps["resources"] == nil || caps["prompts"] == nil || caps["tools"] == nil {
		t.Fatalf("initialize capabilities = %#v, want tools/resources/prompts", caps)
	}

	listedResources := postMCP(t, handler, map[string]any{"jsonrpc": "2.0", "id": 2, "method": "resources/list"})
	resources := listedResources["result"].(map[string]any)["resources"].([]any)
	if !mcpResourcesContain(resources, "goncho://status") || !mcpResourcesContain(resources, "goncho://latest") || mcpResourcesContain(resources, "goncho://recall/prompt") {
		t.Fatalf("resources/list = %#v, want resources without prompt URI", resources)
	}

	readLatest := postMCP(t, handler, map[string]any{"jsonrpc": "2.0", "id": 3, "method": "resources/read", "params": map[string]any{"uri": "goncho://latest", "peer": "mcp-peer", "session_key": "mcp-session", "limit": 1}})
	contents := readLatest["result"].(map[string]any)["contents"].([]any)
	if len(contents) != 1 || !strings.Contains(contents[0].(map[string]any)["text"].(string), "MCP resources expose local evidence") {
		t.Fatalf("resources/read latest = %#v", readLatest)
	}

	listedPrompts := postMCP(t, handler, map[string]any{"jsonrpc": "2.0", "id": 4, "method": "prompts/list"})
	prompts := listedPrompts["result"].(map[string]any)["prompts"].([]any)
	if !mcpPromptsContain(prompts, "recall_prompt") || !mcpPromptsContain(prompts, "session_handoff") || !mcpPromptsContain(prompts, "verification_before_action") {
		t.Fatalf("prompts/list = %#v, want documented Goncho prompts", prompts)
	}

	gotPrompt := postMCP(t, handler, map[string]any{"jsonrpc": "2.0", "id": 5, "method": "prompts/get", "params": map[string]any{"name": "recall_prompt", "arguments": map[string]any{"peer": "mcp-peer", "query": "local evidence", "limit": 2}}})
	messages := gotPrompt["result"].(map[string]any)["messages"].([]any)
	if len(messages) != 1 || !strings.Contains(messages[0].(map[string]any)["content"].(map[string]any)["text"].(string), "Use goncho_recall") {
		t.Fatalf("prompts/get recall_prompt = %#v", gotPrompt)
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

type slowMCPTool struct{ timeout time.Duration }

func (s slowMCPTool) Name() string            { return "slow" }
func (s slowMCPTool) Description() string     { return "slow test tool" }
func (s slowMCPTool) Schema() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (s slowMCPTool) Timeout() time.Duration  { return s.timeout }
func (s slowMCPTool) Execute(ctx context.Context, _ json.RawMessage) (json.RawMessage, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(50 * time.Millisecond):
		return json.RawMessage(`{"ok":true}`), nil
	}
}

func decodeMCPResponses(t *testing.T, text string) []map[string]any {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(text), "\n")
	out := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var response map[string]any
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			t.Fatalf("decode MCP response line: %v\n%s", err, line)
		}
		out = append(out, response)
	}
	return out
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

func mcpResourcesContain(resources []any, uri string) bool {
	for _, resource := range resources {
		obj, ok := resource.(map[string]any)
		if ok && obj["uri"] == uri {
			return true
		}
	}
	return false
}

func mcpPromptsContain(prompts []any, name string) bool {
	for _, prompt := range prompts {
		obj, ok := prompt.(map[string]any)
		if ok && obj["name"] == name {
			return true
		}
	}
	return false
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

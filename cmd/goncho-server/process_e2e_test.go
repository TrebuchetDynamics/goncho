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

	goncho "github.com/TrebuchetDynamics/goncho/service"
)

func TestProcessE2E_GonchoServerOperatorCommandsUseRealCLI(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	bin := filepath.Join(t.TempDir(), "goncho-server-process-e2e")
	build := exec.CommandContext(ctx, "go", "build", "-o", bin, ".")
	if raw, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build ./cmd/goncho-server: %v\n%s", err, raw)
	}

	dir := t.TempDir()
	dbPath := filepath.Join(dir, "goncho.db")
	configPath := filepath.Join(dir, "goncho-server.json")

	var onboarding onboardingReport
	runGonchoServerProcessJSON(t, ctx, bin, &onboarding, "onboarding", "-db", dbPath, "-config", configPath, "-addr", "127.0.0.1:0")
	if onboarding.Status != "plan" || onboarding.Mutates || onboarding.DBPath != dbPath || onboarding.ConfigPath != configPath || !strings.Contains(onboarding.MCPURL, "127.0.0.1:0") {
		t.Fatalf("onboarding = %+v, want non-mutating local plan", onboarding)
	}
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		t.Fatalf("onboarding created db or unexpected stat error: %v", err)
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("onboarding created config or unexpected stat error: %v", err)
	}

	var init initReport
	runGonchoServerProcessJSON(t, ctx, bin, &init, "init", "-db", dbPath, "-config", configPath, "-workspace", "process-workspace", "-observer", "process-observer")
	if init.Status != "ok" || init.DBPath != dbPath || init.ConfigPath != configPath {
		t.Fatalf("init = %+v, want ok with configured paths", init)
	}
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("init did not create db: %v", err)
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("init did not create config: %v", err)
	}

	var health healthReport
	runGonchoServerProcessJSON(t, ctx, bin, &health, "health", "-db", dbPath)
	if health.Status != "ok" || health.DB.Status != "ok" || health.Migrations.Status != "ok" || health.Tools.Count != 6 || !slices.Contains(health.Tools.Available, "goncho_recall") {
		t.Fatalf("health = %+v, want migrated DB and complete public MCP tool surface", health)
	}

	var security goncho.ServerModeSecurityRequirement
	runGonchoServerProcessJSON(t, ctx, bin, &security, "security")
	if security.Mode != "server" || security.Status != goncho.ServerModeStatusRequirementsOnly || security.EnforcementEnabled || !slices.Contains(security.Roles, "admin") || !strings.Contains(security.AuthRequirement, "loopback-only") {
		t.Fatalf("security = %+v, want requirements-only server security contract", security)
	}

	var demo demoReport
	demoDB := filepath.Join(dir, "demo.db")
	runGonchoServerProcessJSON(t, ctx, bin, &demo, "demo", "-db", demoDB)
	if demo.Status != "ok" || !demo.Recall.Proved || !demo.Context.Proved || demo.Recall.SelectedCount == 0 {
		t.Fatalf("demo = %+v, want real-process recall/context proof", demo)
	}

	serveDB := filepath.Join(dir, "public-bind.db")
	serve := exec.CommandContext(ctx, bin, "serve", "-db", serveDB, "-addr", "0.0.0.0:0")
	raw, err := serve.CombinedOutput()
	if err == nil {
		t.Fatalf("unauthenticated non-loopback serve unexpectedly succeeded:\n%s", raw)
	}
	if got := string(raw); !strings.Contains(got, "refusing unauthenticated non-loopback bind") || !strings.Contains(got, "auth token") || !strings.Contains(got, "loopback") {
		t.Fatalf("non-loopback serve output = %q, want auth/loopback guard", got)
	}
	if _, err := os.Stat(serveDB); !os.IsNotExist(err) {
		t.Fatalf("rejected non-loopback serve created db or unexpected stat error: %v", err)
	}
}

func TestProcessE2E_GonchoServerStdioMCPUsesRealCLI(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	bin := filepath.Join(t.TempDir(), "goncho-server-stdio-e2e")
	build := exec.CommandContext(ctx, "go", "build", "-o", bin, ".")
	if raw, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build ./cmd/goncho-server: %v\n%s", err, raw)
	}

	dbPath := filepath.Join(t.TempDir(), "stdio.db")
	stdin := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"goncho_remember","arguments":{"peer_id":"stdio-peer","session_key":"stdio-session","content":"Process stdio MCP remembers the amber narwhal."}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"goncho_search","arguments":{"peer_id":"stdio-peer","session_key":"stdio-session","query":"amber narwhal"}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"goncho_recall","arguments":{"peer_id":"stdio-peer","session_key":"stdio-session","query":"amber narwhal","compact":true}}}`,
	}, "\n") + "\n"

	cmd := exec.CommandContext(ctx, bin, "stdio", "-db", dbPath)
	cmd.Stdin = strings.NewReader(stdin)
	raw, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("goncho-server stdio failed: %v\n%s", err, raw)
	}
	responses := decodeProcessMCPResponses(t, raw)
	if len(responses) != 5 {
		t.Fatalf("stdio responses = %#v, want 5", responses)
	}
	for i, resp := range responses {
		if resp.Error != nil {
			t.Fatalf("stdio response %d error = %+v", i+1, resp.Error)
		}
	}
	out := string(raw)
	for _, want := range []string{"goncho_recall", "amber narwhal", `\"success\":true`, `\"action\":\"search\"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdio output missing %q:\n%s", want, out)
		}
	}
}

func runGonchoServerProcessJSON(t *testing.T, ctx context.Context, bin string, out any, args ...string) {
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

func decodeProcessMCPResponses(t *testing.T, raw []byte) []mcpResponse {
	t.Helper()
	var out []mcpResponse
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var resp mcpResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			t.Fatalf("decode MCP response %q: %v\n%s", line, err, raw)
		}
		out = append(out, resp)
	}
	return out
}

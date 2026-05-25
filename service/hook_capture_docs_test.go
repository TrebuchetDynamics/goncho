package goncho

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestHostHookCapturePolicyDocsNameCapturedIgnoredAndPermissionEvents(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(file), "..", "docs-site", "src", "content", "docs", "integrations", "gormes-agent.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile gormes integration docs: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"| Event | Captured behavior | Ignored by default | Host-specific permission |",
		"prompt",
		"assistant_response",
		"pre_tool_use",
		"post_tool_use",
		"tool_failure",
		"compaction",
		"subagent_start",
		"subagent_stop",
		"stop",
		"session_end",
		"Ignored or adapter-owned events",
		"permission",
		"redacts common secret shapes",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("host hook docs missing marker %q", want)
		}
	}
}

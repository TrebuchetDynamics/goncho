package docs_test

import (
	"os"
	"strings"
	"testing"
)

func TestSecurityDocsCoverLocalFirstThreats(t *testing.T) {
	raw, err := os.ReadFile("../SECURITY.md")
	if err != nil {
		t.Fatalf("read SECURITY.md: %v", err)
	}
	doc := strings.ToLower(string(raw))
	for _, want := range []string{"vulnerability reporting", "supported versions", "local files", "non-loopback", "prompt injection", "quarantine", "redaction", "snapshot exports", "do not expose directly to the internet"} {
		if !strings.Contains(doc, want) {
			t.Fatalf("SECURITY.md missing %q", want)
		}
	}
}

package docs_test

import (
	"strings"
	"testing"
)

func TestSecurityDocsCoverLocalFirstThreats(t *testing.T) {
	doc := strings.ToLower(mustReadGuardFile(t, "../../SECURITY.md"))
	for _, want := range []string{"vulnerability reporting", "supported versions", "local files", "non-loopback", "prompt injection", "quarantine", "redaction", "snapshot exports", "do not expose directly to the internet"} {
		if !strings.Contains(doc, want) {
			t.Fatalf("SECURITY.md missing %q", want)
		}
	}
}

package docs_test

import "testing"

func TestSecurityDocsCoverLocalFirstThreats(t *testing.T) {
	mustContainAllFold(t, mustReadGuardFile(t, "../../SECURITY.md"), "SECURITY.md", []string{"vulnerability reporting", "supported versions", "local files", "non-loopback", "prompt injection", "quarantine", "redaction", "snapshot exports", "do not expose directly to the internet"})
}

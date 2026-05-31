package operations_test

import (
	"testing"

	"github.com/TrebuchetDynamics/goncho/docs/guards/guardtest"
)

func TestSecurityDocsCoverLocalFirstThreats(t *testing.T) {
	guardtest.ContainsAllFold(t, guardtest.ReadRepoFile(t, "SECURITY.md"), "SECURITY.md", []string{"vulnerability reporting", "supported versions", "local files", "non-loopback", "prompt injection", "quarantine", "redaction", "snapshot exports", "do not expose directly to the internet"})
}

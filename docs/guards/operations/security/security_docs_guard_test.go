package operations_test

import (
	"testing"

	"github.com/TrebuchetDynamics/goncho/docs/guards/operations/opguard"
)

func TestSecurityDocsCoverLocalFirstThreats(t *testing.T) {
	opguard.VerifyFileContract(t, opguard.FileContract{
		Path:  "SECURITY.md",
		Label: "SECURITY.md",
		Markers: []string{
			"vulnerability reporting", "supported versions", "local files", "non-loopback",
			"prompt injection", "quarantine", "redaction", "snapshot exports",
			opguard.InternetExposureWarning,
		},
		Fold: true,
	})
}

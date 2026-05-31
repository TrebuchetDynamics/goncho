package operations_test

import (
	"testing"

	"github.com/TrebuchetDynamics/goncho/docs/guards/operations/opguard"
)

func TestServerModeOperationalPackagingFilesStayLocalFirst(t *testing.T) {
	opguard.VerifyFileContracts(t, []opguard.FileContract{
		{Path: "Dockerfile", Markers: []string{"goncho-server", "EXPOSE 8765", "HEALTHCHECK", "127.0.0.1:8765"}},
		{Path: "docker-compose.yml", Markers: []string{"goncho-server", "127.0.0.1:8765:8765", "-auth-token", "goncho-data:/data"}},
		{Path: "scripts/docker_compose_smoke.py", Label: "docker compose smoke wrapper", Markers: []string{"export_module", "smoke.docker_compose"}},
		{Path: "scripts/smoke/compose/docker_compose.py", Label: "docker compose smoke", Markers: []string{"docker", "compose", "up", "/health", "demo", "down", "-v"}},
		{Path: "Makefile", Markers: []string{"docker-compose-smoke:", "scripts/docker_compose_smoke.py"}},
	})
}

func TestDeploymentDocsNameConservativeTargetAndBackupRestore(t *testing.T) {
	opguard.VerifyFileContract(t, opguard.FileContract{
		Path:  "docs/deployment-local-shared-service.md",
		Label: "deployment doc",
		Markers: []string{
			"local shared service", "loopback", "auth token", "docker compose", "health",
			"snapshot manifest", "export", "restore", opguard.InternetExposureWarning,
		},
		Fold: true,
	})
}

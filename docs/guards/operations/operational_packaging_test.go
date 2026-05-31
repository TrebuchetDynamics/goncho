package operations_test

import (
	"testing"

	"github.com/TrebuchetDynamics/goncho/docs/guards/guardtest"
)

func TestServerModeOperationalPackagingFilesStayLocalFirst(t *testing.T) {
	guardtest.ContainsAll(t, guardtest.ReadRepoFile(t, "Dockerfile"), "Dockerfile", []string{"goncho-server", "EXPOSE 8765", "HEALTHCHECK", "127.0.0.1:8765"})
	guardtest.ContainsAll(t, guardtest.ReadRepoFile(t, "docker-compose.yml"), "docker-compose.yml", []string{"goncho-server", "127.0.0.1:8765:8765", "-auth-token", "goncho-data:/data"})
	guardtest.ContainsAll(t, guardtest.ReadRepoFile(t, "scripts/docker_compose_smoke.py"), "docker compose smoke wrapper", []string{"export_module", "smoke.docker_compose"})
	guardtest.ContainsAll(t, guardtest.ReadRepoFile(t, "scripts/smoke/compose/docker_compose.py"), "docker compose smoke", []string{"docker", "compose", "up", "/health", "demo", "down", "-v"})
	guardtest.ContainsAll(t, guardtest.ReadRepoFile(t, "Makefile"), "Makefile", []string{"docker-compose-smoke:", "scripts/docker_compose_smoke.py"})
}

func TestDeploymentDocsNameConservativeTargetAndBackupRestore(t *testing.T) {
	guardtest.ContainsAllFold(t, guardtest.ReadRepoFile(t, "docs/deployment-local-shared-service.md"), "deployment doc", []string{"local shared service", "loopback", "auth token", "docker compose", "health", "snapshot manifest", "export", "restore", "do not expose directly to the internet"})
}

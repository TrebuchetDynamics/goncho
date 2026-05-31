package docs_test

import "testing"

func TestServerModeOperationalPackagingFilesStayLocalFirst(t *testing.T) {
	mustContainAll(t, mustReadGuardFile(t, "../../Dockerfile"), "Dockerfile", []string{"goncho-server", "EXPOSE 8765", "HEALTHCHECK", "127.0.0.1:8765"})
	mustContainAll(t, mustReadGuardFile(t, "../../docker-compose.yml"), "docker-compose.yml", []string{"goncho-server", "127.0.0.1:8765:8765", "-auth-token", "goncho-data:/data"})
	mustContainAll(t, mustReadGuardFile(t, "../../scripts/docker_compose_smoke.py"), "docker compose smoke wrapper", []string{"export_module", "smoke.docker_compose"})
	mustContainAll(t, mustReadGuardFile(t, "../../scripts/smoke/docker_compose.py"), "docker compose smoke", []string{"docker", "compose", "up", "/health", "demo", "down", "-v"})
	mustContainAll(t, mustReadGuardFile(t, "../../Makefile"), "Makefile", []string{"docker-compose-smoke:", "scripts/docker_compose_smoke.py"})
}

func TestDeploymentDocsNameConservativeTargetAndBackupRestore(t *testing.T) {
	mustContainAllFold(t, mustReadGuardFile(t, "../deployment-local-shared-service.md"), "deployment doc", []string{"local shared service", "loopback", "auth token", "docker compose", "health", "snapshot manifest", "export", "restore", "do not expose directly to the internet"})
}

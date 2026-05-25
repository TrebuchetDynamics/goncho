package docs_test

import (
	"os"
	"strings"
	"testing"
)

func TestServerModeOperationalPackagingFilesStayLocalFirst(t *testing.T) {
	dockerfile := mustReadDocTestFile(t, "../Dockerfile")
	for _, want := range []string{"goncho-server", "EXPOSE 8765", "HEALTHCHECK", "127.0.0.1:8765"} {
		if !strings.Contains(dockerfile, want) {
			t.Fatalf("Dockerfile missing %q", want)
		}
	}
	compose := mustReadDocTestFile(t, "../docker-compose.yml")
	for _, want := range []string{"goncho-server", "127.0.0.1:8765:8765", "-auth-token", "goncho-data:/data"} {
		if !strings.Contains(compose, want) {
			t.Fatalf("docker-compose.yml missing %q", want)
		}
	}
	smoke := mustReadDocTestFile(t, "../scripts/docker_compose_smoke.py")
	for _, want := range []string{"docker", "compose", "up", "/health", "demo", "down", "-v"} {
		if !strings.Contains(smoke, want) {
			t.Fatalf("docker compose smoke missing %q", want)
		}
	}
	makefile := mustReadDocTestFile(t, "../Makefile")
	if !strings.Contains(makefile, "docker-compose-smoke:") || !strings.Contains(makefile, "scripts/docker_compose_smoke.py") {
		t.Fatalf("Makefile missing docker-compose-smoke target")
	}
}

func TestDeploymentDocsNameConservativeTargetAndBackupRestore(t *testing.T) {
	doc := strings.ToLower(mustReadDocTestFile(t, "deployment-local-shared-service.md"))
	for _, want := range []string{"local shared service", "loopback", "auth token", "docker compose", "health", "snapshot manifest", "export", "restore", "do not expose directly to the internet"} {
		if !strings.Contains(doc, want) {
			t.Fatalf("deployment doc missing %q", want)
		}
	}
}

func mustReadDocTestFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(raw)
}

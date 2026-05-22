package goncho

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

func TestChangelogReleaseHeadingsHaveMatchingTags(t *testing.T) {
	if _, err := exec.Command("git", "rev-parse", "--show-toplevel").Output(); err != nil {
		t.Skipf("git checkout unavailable: %v", err)
	}

	changelog, err := os.ReadFile("CHANGELOG.md")
	if err != nil {
		t.Fatalf("ReadFile CHANGELOG.md: %v", err)
	}

	releaseHeading := regexp.MustCompile(`(?m)^## (v\d+\.\d+\.\d+) - `)
	matches := releaseHeading.FindAllStringSubmatch(string(changelog), -1)
	if len(matches) == 0 {
		t.Fatal("CHANGELOG.md has no tagged release headings")
	}

	var missing []string
	for _, match := range matches {
		version := match[1]
		out, err := exec.Command("git", "tag", "-l", version).Output()
		if err != nil {
			t.Fatalf("git tag -l %s: %v", version, err)
		}
		if strings.TrimSpace(string(out)) != version {
			missing = append(missing, version)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("CHANGELOG.md release headings without matching git tags: %s", strings.Join(missing, ", "))
	}
}

func TestReleaseSmokeDocsMentionMetadataGuard(t *testing.T) {
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/start/quick-start.md",
		"docs-site/src/content/docs/operators/runbook.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			if !strings.Contains(text, "make release-smoke") {
				t.Fatalf("%s does not mention make release-smoke", path)
			}
			if !strings.Contains(strings.ToLower(text), "release metadata") {
				t.Fatalf("%s does not mention release metadata checks in release-smoke guidance", path)
			}
		})
	}
}

func TestPublicDocsLinkGoReference(t *testing.T) {
	const goReferenceURL = "https://pkg.go.dev/github.com/TrebuchetDynamics/goncho"
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/index.md",
		"docs-site/src/content/docs/start/current-capabilities.md",
		"docs-site/src/content/docs/start/quick-start.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			if !strings.Contains(string(raw), goReferenceURL) {
				t.Fatalf("%s does not link the public Go reference at %s", path, goReferenceURL)
			}
		})
	}
}

func TestPublicDocsMentionEcosystemSmoke(t *testing.T) {
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/index.md",
		"docs-site/src/content/docs/operators/runbook.md",
		"docs-site/src/content/docs/start/current-capabilities.md",
		"docs-site/src/content/docs/start/quick-start.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			if !strings.Contains(string(raw), "make ecosystem-smoke") {
				t.Fatalf("%s does not mention make ecosystem-smoke", path)
			}
		})
	}
}

func TestPublicAdoptionDocsMentionPublicModuleSmoke(t *testing.T) {
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/index.md",
		"docs-site/src/content/docs/start/current-capabilities.md",
		"docs-site/src/content/docs/start/quick-start.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			if !strings.Contains(string(raw), "make public-module-smoke") {
				t.Fatalf("%s does not mention make public-module-smoke", path)
			}
		})
	}
}

func TestPublicDocsMentionLatestReleaseVersion(t *testing.T) {
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/index.md",
		"docs-site/src/content/docs/start/current-capabilities.md",
		"docs-site/src/content/docs/start/quick-start.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			if !strings.Contains(string(raw), "v0.1.0") {
				t.Fatalf("%s does not mention current public release v0.1.0", path)
			}
		})
	}
}

func TestPublicDocsFrameRootModuleAsLibrary(t *testing.T) {
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/index.md",
		"docs-site/src/content/docs/start/current-capabilities.md",
		"docs-site/src/content/docs/start/quick-start.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			if !strings.Contains(string(raw), "root module is a library package") {
				t.Fatalf("%s does not frame the root module as a library package", path)
			}
		})
	}
}

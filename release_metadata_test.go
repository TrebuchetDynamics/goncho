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

func TestPublicDocsLinkRetrievalBenchmarksReference(t *testing.T) {
	wantByPath := map[string][]string{
		"README.md": {
			"Retrieval Benchmarks",
			"docs-site/src/content/docs/reference/retrieval-benchmarks.md",
		},
		"docs-site/src/content/docs/index.md": {
			"Retrieval Benchmarks",
			"/reference/retrieval-benchmarks/",
		},
		"docs-site/src/content/docs/start/current-capabilities.md": {
			"Retrieval Benchmarks",
			"/reference/retrieval-benchmarks/",
		},
		"docs-site/src/content/docs/start/quick-start.md": {
			"Retrieval Benchmarks",
			"/reference/retrieval-benchmarks/",
		},
	}
	for path, wants := range wantByPath {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			text := string(raw)
			for _, want := range wants {
				if !strings.Contains(text, want) {
					t.Fatalf("%s does not link benchmark methodology reference marker %q", path, want)
				}
			}
		})
	}
}

func TestPublicDocsSurfaceExternalAdapterContract(t *testing.T) {
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
			text := string(raw)
			for _, want := range []string{"external adapter contract", "agentmemory PR #583"} {
				if !strings.Contains(text, want) {
					t.Fatalf("%s does not surface external adapter credibility marker %q", path, want)
				}
			}
		})
	}
}

func TestPublicDocsMentionBackendComparisonSmoke(t *testing.T) {
	const smokeCommand = "make bench-locomo-backends-smoke"
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
			if !strings.Contains(string(raw), smokeCommand) {
				t.Fatalf("%s does not mention external backend comparison smoke command %q", path, smokeCommand)
			}
		})
	}
}

func TestBenchmarkDocsMentionConversationScopedBackendComparison(t *testing.T) {
	const marker = "conversation-scoped"
	for _, path := range []string{
		"README.md",
		"docs-site/src/content/docs/reference/retrieval-benchmarks.md",
		"docs-site/src/content/docs/operators/runbook.md",
		"docs/benchmarks/external-backend-adapters.md",
	} {
		t.Run(path, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			if !strings.Contains(strings.ToLower(string(raw)), marker) {
				t.Fatalf("%s does not mention %q backend comparison", path, marker)
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
			if !strings.Contains(string(raw), "v0.1.1") {
				t.Fatalf("%s does not mention current public release v0.1.1", path)
			}
		})
	}
}

func TestPublicDocsUseLatestQualifiedGoGet(t *testing.T) {
	const qualifiedGoGet = "go get github.com/TrebuchetDynamics/goncho@latest"
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
			if !strings.Contains(string(raw), qualifiedGoGet) {
				t.Fatalf("%s does not mention version-qualified library adoption command %q", path, qualifiedGoGet)
			}
		})
	}
}

func TestPublicDocsMentionPublishedReleaseDate(t *testing.T) {
	const publishedDate = "published May 22, 2026"
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
			if !strings.Contains(string(raw), publishedDate) {
				t.Fatalf("%s does not mention public release date %q", path, publishedDate)
			}
		})
	}
}

func TestPublicDocsWarnRootGoInstallIsUnsupported(t *testing.T) {
	const rootInstallWarning = "not a root `go install` target"
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
			if !strings.Contains(string(raw), rootInstallWarning) {
				t.Fatalf("%s does not warn that the public module is %q", path, rootInstallWarning)
			}
		})
	}
}

func TestReadmeSurfacesPkgGoDevEvaluationPath(t *testing.T) {
	raw, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatalf("ReadFile README.md: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"## At a Glance",
		"If you are evaluating Goncho on pkg.go.dev",
		"First useful call",
		"compiled `NewService` example",
		"compiled `Service.Context` example",
		"compiled `Service.Search` example",
		"What to read next",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("README.md does not surface pkg.go.dev evaluation marker %q", want)
		}
	}
}

func TestPackageDocsIncludeCompiledNewServiceExample(t *testing.T) {
	raw, err := os.ReadFile("example_service_test.go")
	if err != nil {
		t.Fatalf("ReadFile example_service_test.go: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"func ExampleNewService()",
		"memory.OpenSqlite",
		"goncho.RunMigrations",
		"goncho.NewService",
		"svc.SetProfile",
		"svc.Profile",
		"// Output:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("example_service_test.go does not include pkg.go.dev example marker %q", want)
		}
	}
}

func TestPackageDocsIncludeCompiledContextExample(t *testing.T) {
	raw, err := os.ReadFile("example_service_test.go")
	if err != nil {
		t.Fatalf("ReadFile example_service_test.go: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"func ExampleService_Context()",
		"svc.Conclude",
		"svc.Context",
		"orientation.Representation",
		"Current conclusions:",
		"// Output:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("example_service_test.go does not include pkg.go.dev context example marker %q", want)
		}
	}
}

func TestPackageDocsIncludeCompiledSearchExample(t *testing.T) {
	raw, err := os.ReadFile("example_service_test.go")
	if err != nil {
		t.Fatalf("ReadFile example_service_test.go: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"func ExampleService_Search()",
		"svc.Conclude",
		"svc.Search",
		"results.Results[0].Source",
		"results.Results[0].Content",
		"// Output:",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("example_service_test.go does not include pkg.go.dev search example marker %q", want)
		}
	}
}

func TestReleaseMetadataSmokeIncludesReadmePkgGoDevGuard(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"ReadmeSurfacesPkgGoDevEvaluationPath",
		"ReleaseMetadataSmokeIncludesReadmePkgGoDevGuard",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-metadata-smoke does not include README pkg.go.dev guard %q", want)
		}
	}
}

func TestReleaseMetadataSmokeIncludesPackageExampleGuard(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"PackageDocsIncludeCompiledNewServiceExample",
		"ReleaseMetadataSmokeIncludesPackageExampleGuard",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-metadata-smoke does not include package example guard %q", want)
		}
	}
}

func TestReleaseMetadataSmokeIncludesContextExampleGuard(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"PackageDocsIncludeCompiledContextExample",
		"ReleaseMetadataSmokeIncludesContextExampleGuard",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-metadata-smoke does not include context example guard %q", want)
		}
	}
}

func TestReleaseMetadataSmokeIncludesSearchExampleGuard(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"PackageDocsIncludeCompiledSearchExample",
		"ReleaseMetadataSmokeIncludesSearchExampleGuard",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-metadata-smoke does not include search example guard %q", want)
		}
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

func TestEcosystemSmokeIncludesPublicReleaseMetadata(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"public-release-smoke:",
		"$(MAKE) public-release-smoke",
		"go list -m -json github.com/TrebuchetDynamics/goncho@latest",
		`"Version":`,
		`"Time":`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("Makefile does not include public release metadata smoke marker %q", want)
		}
	}
}

func TestPublicReleaseSmokeChecksDocumentedLatestMetadata(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"PUBLIC_LATEST_VERSION := v0.1.1",
		"PUBLIC_LATEST_PUBLISHED_DATE := 2026-05-22",
		`"Version": "$(PUBLIC_LATEST_VERSION)"`,
		`"Time": "$(PUBLIC_LATEST_PUBLISHED_DATE)`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("Makefile does not pin public release metadata smoke marker %q", want)
		}
	}
}

func TestPublicDocsExplainDocumentedLatestPublicReleaseSmoke(t *testing.T) {
	const marker = "documented public `@latest` version and published date"
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
			if !strings.Contains(string(raw), marker) {
				t.Fatalf("%s does not explain public-release smoke marker %q", path, marker)
			}
		})
	}
}

func TestLocalModuleSmokeChecksGoModMetadata(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"local-module-smoke:",
		"$(MAKE) local-module-smoke",
		"go list -m -json",
		`"Path": "github.com/TrebuchetDynamics/goncho"`,
		`"GoVersion":`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("Makefile does not include local go.mod metadata smoke marker %q", want)
		}
	}
}

func TestPackageDocSurfacesPkgGoDevLandingContent(t *testing.T) {
	out, err := exec.Command("go", "doc", ".").Output()
	if err != nil {
		t.Fatalf("go doc .: %v", err)
	}
	text := strings.Join(strings.Fields(string(out)), " ")
	for _, want := range []string{
		"Use Goncho when",
		"Quick start",
		"verification before action",
		"goncho.NewService",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("go doc . output does not include pkg.go.dev landing marker %q\n%s", want, text)
		}
	}
}

func TestPackageDocPointsPkgGoDevReadersToCompiledExamples(t *testing.T) {
	out, err := exec.Command("go", "doc", ".").Output()
	if err != nil {
		t.Fatalf("go doc .: %v", err)
	}
	text := strings.Join(strings.Fields(string(out)), " ")
	for _, want := range []string{
		"pkg.go.dev examples",
		"ExampleNewService",
		"ExampleService_Context",
		"ExampleService_Search",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("go doc . output does not point pkg.go.dev readers to compiled examples marker %q\n%s", want, text)
		}
	}
}

func TestReleaseMetadataSmokeIncludesPackageDocExamplesGuard(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"PackageDocPointsPkgGoDevReadersToCompiledExamples",
		"ReleaseMetadataSmokeIncludesPackageDocExamplesGuard",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("release-metadata-smoke does not include package doc examples guard %q", want)
		}
	}
}

func TestPackageDocSmokeChecksLocalGoDoc(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"package-doc-smoke:",
		"$(MAKE) package-doc-smoke",
		"go doc . >/dev/null",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("Makefile does not include package documentation smoke marker %q", want)
		}
	}
}

func TestDocsSiteSmokeBuildsPublicDocs(t *testing.T) {
	raw, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatalf("ReadFile Makefile: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"docs-site-smoke:",
		"$(MAKE) docs-site-smoke",
		"cd docs-site && npm run build",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("Makefile does not include docs-site smoke marker %q", want)
		}
	}
}

func TestPublicDocsMentionDocsSiteSmoke(t *testing.T) {
	const smokeCommand = "make docs-site-smoke"
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
			if !strings.Contains(string(raw), smokeCommand) {
				t.Fatalf("%s does not mention docs-site smoke command %q", path, smokeCommand)
			}
		})
	}
}

func TestPublicDocsMentionPackageDocSmoke(t *testing.T) {
	const smokeCommand = "make package-doc-smoke"
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
			if !strings.Contains(string(raw), smokeCommand) {
				t.Fatalf("%s does not mention package documentation smoke command %q", path, smokeCommand)
			}
		})
	}
}

func TestPublicDocsMentionLocalModuleSmoke(t *testing.T) {
	const smokeCommand = "make local-module-smoke"
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
			if !strings.Contains(string(raw), smokeCommand) {
				t.Fatalf("%s does not mention local go.mod metadata smoke command %q", path, smokeCommand)
			}
		})
	}
}

func TestPublicDocsMentionPublicReleaseSmoke(t *testing.T) {
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
			if !strings.Contains(string(raw), "make public-release-smoke") {
				t.Fatalf("%s does not mention make public-release-smoke", path)
			}
		})
	}
}

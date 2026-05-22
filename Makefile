LONGMEMEVAL_REVISION := 98d7416c24c778c2fee6e6f3006e7a073259d48f
LONGMEMEVAL_SHA256 := d6f21ea9d60a0d56f34a05b609c79c88a451d2ae03597821ea3d5a9678c3a442
LONGMEMEVAL_DATE := $(shell date -u +%Y-%m-%d)
BENCH_SYSTEMS := random bm25 sqlite-fts5 goncho-no-rank goncho
LONGMEMEVAL_RUNS ?= 20
LOCOMO_DATE := $(shell date -u +%Y-%m-%d)
LOCOMO_SMOKE_MEMORIES := ./cmd/goncho-bench/testdata/locomo-smoke/memories.jsonl
LOCOMO_SMOKE_QUESTIONS := ./cmd/goncho-bench/testdata/locomo-smoke/questions.jsonl
LOCOMO_MEMORIES := ./data/locomo/memories.jsonl
LOCOMO_QUESTIONS := ./data/locomo/questions.jsonl
PUBLIC_LATEST_VERSION := v0.1.1
PUBLIC_LATEST_PUBLISHED_DATE := 2026-05-22

.PHONY: release-smoke release-metadata-smoke ecosystem-smoke public-release-smoke local-module-smoke package-doc-smoke docs-site-smoke public-module-smoke install-smoke bench-longmemeval-s-smoke bench-longmemeval-s prepare-longmemeval-s bench-locomo-smoke bench-locomo bench-locomo-backends-smoke bench-locomo-backends

release-smoke:
	$(MAKE) release-metadata-smoke
	$(MAKE) ecosystem-smoke
	go test ./...
	go vet ./...
	go test -race ./...

release-metadata-smoke:
	go test . -run 'Test(ChangelogReleaseHeadingsHaveMatchingTags|ReleaseSmokeDocsMentionMetadataGuard|PublicDocsLinkGoReference|PublicDocsMentionEcosystemSmoke|PublicDocsLinkRetrievalBenchmarksReference|PublicDocsSurfaceExternalAdapterContract|PublicDocsMentionBackendComparisonSmoke|BenchmarkDocsMentionConversationScopedBackendComparison|PublicAdoptionDocsMentionPublicModuleSmoke|PublicDocsMentionLatestReleaseVersion|PublicDocsUseLatestQualifiedGoGet|PublicDocsMentionPublishedReleaseDate|PublicDocsWarnRootGoInstallIsUnsupported|ReadmeSurfacesPkgGoDevEvaluationPath|ReadmeSurfacesTrustBoundaryGuide|ReadmeSurfacesPkgGoDevAPIMap|ReadmeSurfacesImportPathGuide|ReadmeSurfacesHostIntegrationChecklist|ReadmeSurfacesGoDevSignalMap|ReadmeSurfacesVersioningAndAdoptionNotes|ReadmeSurfacesMinimalEmbeddedSkeleton|PackageDocsIncludeCompiledNewServiceExample|PackageDocsIncludeCompiledContextExample|PackageDocsIncludeCompiledSearchExample|ReleaseMetadataSmokeIncludesReadmePkgGoDevGuard|ReleaseMetadataSmokeIncludesReadmeTrustBoundaryGuard|ReleaseMetadataSmokeIncludesReadmeAPIMapGuard|ReleaseMetadataSmokeIncludesReadmeImportPathGuard|ReleaseMetadataSmokeIncludesReadmeHostIntegrationGuard|ReleaseMetadataSmokeIncludesReadmeGoDevSignalGuard|ReleaseMetadataSmokeIncludesReadmeVersioningGuard|ReleaseMetadataSmokeIncludesReadmeMinimalSkeletonGuard|ReleaseMetadataSmokeIncludesPackageExampleGuard|ReleaseMetadataSmokeIncludesContextExampleGuard|ReleaseMetadataSmokeIncludesSearchExampleGuard|PublicDocsFrameRootModuleAsLibrary|EcosystemSmokeIncludesPublicReleaseMetadata|PublicReleaseSmokeChecksDocumentedLatestMetadata|PublicDocsExplainDocumentedLatestPublicReleaseSmoke|LocalModuleSmokeChecksGoModMetadata|PackageDocPointsPkgGoDevReadersToCompiledExamples|PackageDocSurfacesInstallAndCommandBoundary|PackageDocSurfacesImportPathGuide|PackageDocSurfacesTrustBoundaryGuide|PackageDocSurfacesHostIntegrationChecklist|PackageDocSurfacesPrimaryAPIPath|PackageDocSurfacesGoDevPackageSignals|PackageDocSurfacesVersioningAndAdoptionNotes|ReleaseMetadataSmokeIncludesPackageDocExamplesGuard|ReleaseMetadataSmokeIncludesPackageDocInstallGuard|ReleaseMetadataSmokeIncludesPackageDocImportPathGuard|ReleaseMetadataSmokeIncludesPackageDocTrustBoundaryGuard|ReleaseMetadataSmokeIncludesPackageDocHostIntegrationGuard|ReleaseMetadataSmokeIncludesPackageDocAPIPathGuard|ReleaseMetadataSmokeIncludesPackageDocGoDevSignalGuard|ReleaseMetadataSmokeIncludesPackageDocVersioningGuard|PackageDocSmokeChecksLocalGoDoc|DocsSiteSmokeBuildsPublicDocs|PublicDocsMentionDocsSiteSmoke|PublicDocsMentionPackageDocSmoke|PublicDocsMentionLocalModuleSmoke|PublicDocsMentionPublicReleaseSmoke)' -count=1

ecosystem-smoke:
	$(MAKE) public-release-smoke
	$(MAKE) local-module-smoke
	$(MAKE) package-doc-smoke
	$(MAKE) docs-site-smoke
	$(MAKE) public-module-smoke
	$(MAKE) install-smoke

public-release-smoke:
	@info=$$(go list -m -json github.com/TrebuchetDynamics/goncho@latest); \
	printf '%s\n' "$$info"; \
	printf '%s\n' "$$info" | grep -Fq '"Version": "$(PUBLIC_LATEST_VERSION)"'; \
	printf '%s\n' "$$info" | grep -Fq '"Time": "$(PUBLIC_LATEST_PUBLISHED_DATE)'

local-module-smoke:
	@info=$$(go list -m -json); \
	printf '%s\n' "$$info"; \
	printf '%s\n' "$$info" | grep -q '"Path": "github.com/TrebuchetDynamics/goncho"'; \
	printf '%s\n' "$$info" | grep -q '"GoVersion":'

package-doc-smoke:
	go doc . >/dev/null

docs-site-smoke:
	cd docs-site && npm run build

public-module-smoke:
	@tmp=$$(mktemp -d); \
	echo "verifying public github.com/TrebuchetDynamics/goncho@latest import in $$tmp"; \
	cd "$$tmp"; \
	go mod init goncho-public-module-smoke >/dev/null; \
	go get github.com/TrebuchetDynamics/goncho@latest; \
	printf '%s\n' \
		'package main' \
		'' \
		'import (' \
		'    "database/sql"' \
		'    "fmt"' \
		'    "github.com/TrebuchetDynamics/goncho"' \
		')' \
		'' \
		'func main() {' \
		'    var db *sql.DB' \
		'    if goncho.NewService(db, goncho.Config{}, nil) == nil {' \
		'        panic("nil service")' \
		'    }' \
		'    fmt.Println("goncho public module import ok")' \
		'}' > main.go; \
	go run .

install-smoke:
	@tmp=$$(mktemp -d); \
	echo "installing ./cmd/goncho-bench to $$tmp"; \
	GOBIN="$$tmp" go install ./cmd/goncho-bench; \
	"$$tmp/goncho-bench" --help >/dev/null

bench-longmemeval-s-smoke:
	@mkdir -p artifacts/bench-smoke docs/benchmarks/results docs/benchmarks/failures
	@for system in $(BENCH_SYSTEMS); do \
		go run ./cmd/goncho-bench \
			--dataset ./cmd/goncho-bench/testdata/tiny-longmemeval.jsonl \
			--out ./docs/benchmarks/results/longmemeval-s-smoke-$$system.json \
			--failures ./docs/benchmarks/failures/longmemeval-s-smoke-$$system.jsonl \
			--db ./artifacts/bench-smoke/$$system.db \
			--system $$system \
			--dataset-revision smoke-fixture \
			--dataset-sha256 smoke-fixture \
			--limit 10 \
			--runs 2; \
	done

prepare-longmemeval-s:
	python3 ./scripts/prepare_longmemeval_s.py \
		--raw-dir ./artifacts/longmemeval/raw \
		--out ./artifacts/longmemeval/longmemeval-s-goncho.jsonl

bench-longmemeval-s: prepare-longmemeval-s
	@mkdir -p artifacts/longmemeval docs/benchmarks/results docs/benchmarks/failures
	@for system in $(BENCH_SYSTEMS); do \
		go run ./cmd/goncho-bench \
			--dataset ./artifacts/longmemeval/longmemeval-s-goncho.jsonl \
			--out ./docs/benchmarks/results/longmemeval-s-$(LONGMEMEVAL_DATE)-$$system.json \
			--failures ./docs/benchmarks/failures/longmemeval-s-$(LONGMEMEVAL_DATE)-$$system.jsonl \
			--db ./artifacts/longmemeval/$$system.db \
			--system $$system \
			--dataset-revision $(LONGMEMEVAL_REVISION) \
			--dataset-sha256 $(LONGMEMEVAL_SHA256) \
			--limit 10 \
			--runs $(LONGMEMEVAL_RUNS); \
	done

bench-locomo-smoke:
	@mkdir -p docs/benchmarks/results docs/benchmarks/failures
	go run ./cmd/goncho-bench \
		--locomo-name "LOCOMO smoke" \
		--locomo-memories $(LOCOMO_SMOKE_MEMORIES) \
		--locomo-questions $(LOCOMO_SMOKE_QUESTIONS) \
		--out ./docs/benchmarks/results/locomo-smoke-goncho.json \
		--failures ./docs/benchmarks/failures/locomo-smoke-categories.jsonl \
		--locomo-md-out ./docs/benchmarks/locomo-smoke.md

prepare-locomo:
	python3 ./scripts/prepare_locomo.py \
		--raw-dir ./data/locomo/raw \
		--out-dir ./data/locomo

bench-locomo: prepare-locomo
	@mkdir -p docs/benchmarks/results docs/benchmarks/failures
	go run ./cmd/goncho-bench \
		--locomo-name "LOCOMO" \
		--locomo-memories $(LOCOMO_MEMORIES) \
		--locomo-questions $(LOCOMO_QUESTIONS) \
		--out ./docs/benchmarks/results/locomo-$(LOCOMO_DATE)-goncho.json \
		--failures ./docs/benchmarks/failures/locomo-$(LOCOMO_DATE)-categories.jsonl \
		--locomo-md-out ./docs/benchmarks/locomo-$(LOCOMO_DATE).md

bench-locomo-backends-smoke:
	@mkdir -p artifacts/locomo-backends docs/benchmarks/results docs/benchmarks/failures
	python3 ./scripts/bench_agentmemory_locomo.py --memories $(LOCOMO_SMOKE_MEMORIES) --questions $(LOCOMO_SMOKE_QUESTIONS) --out ./artifacts/locomo-backends/agentmemory-smoke.jsonl
	python3 ./scripts/bench_mem0_locomo.py --memories $(LOCOMO_SMOKE_MEMORIES) --questions $(LOCOMO_SMOKE_QUESTIONS) --out ./artifacts/locomo-backends/mem0-smoke.jsonl
	go run ./cmd/goncho-bench \
		--locomo-memories $(LOCOMO_SMOKE_MEMORIES) \
		--locomo-questions $(LOCOMO_SMOKE_QUESTIONS) \
		--locomo-agentmemory-results ./artifacts/locomo-backends/agentmemory-smoke.jsonl \
		--locomo-mem0-results ./artifacts/locomo-backends/mem0-smoke.jsonl \
		--locomo-backend-comparison-json-out ./docs/benchmarks/results/locomo-backend-comparison-smoke.json \
		--locomo-backend-comparison-failures-out ./docs/benchmarks/failures/locomo-backend-comparison-smoke.jsonl \
		--locomo-backend-comparison-md-out ./docs/benchmarks/locomo-backend-comparison-smoke.md

bench-locomo-backends: prepare-locomo
	@mkdir -p artifacts/locomo-backends docs/benchmarks/results docs/benchmarks/failures
	python3 ./scripts/bench_agentmemory_locomo.py --memories $(LOCOMO_MEMORIES) --questions $(LOCOMO_QUESTIONS) --out ./artifacts/locomo-backends/agentmemory.jsonl
	python3 ./scripts/bench_mem0_locomo.py --memories $(LOCOMO_MEMORIES) --questions $(LOCOMO_QUESTIONS) --out ./artifacts/locomo-backends/mem0.jsonl
	go run ./cmd/goncho-bench \
		--locomo-memories $(LOCOMO_MEMORIES) \
		--locomo-questions $(LOCOMO_QUESTIONS) \
		--locomo-agentmemory-results ./artifacts/locomo-backends/agentmemory.jsonl \
		--locomo-mem0-results ./artifacts/locomo-backends/mem0.jsonl \
		--locomo-backend-comparison-json-out ./docs/benchmarks/results/locomo-backend-comparison.json \
		--locomo-backend-comparison-failures-out ./docs/benchmarks/failures/locomo-backend-comparison.jsonl \
		--locomo-backend-comparison-md-out ./docs/benchmarks/locomo-backend-comparison.md

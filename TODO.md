# Goncho TODO

## Release state

- 2026-05-22: LOCOMO smoke/full retrieval reports now honor the configured top-K limit.
  - Evidence target: `go test ./cmd/goncho-bench -run TestRunLocomoBenchmarkHonorsConfiguredLimit -count=1` proves `--limit 1` reaches the LOCOMO retrieval report path and caps each local system's reported retrieved IDs.
  - Result: reproducible LOCOMO retrieval reports now use the operator-requested top-K window instead of always evaluating every local system with top 10.

- 2026-05-22: LOCOMO backend comparison now honors the configured top-K limit in the full report path.
  - Evidence target: `go test ./cmd/goncho-bench -run 'Test(RunLocomoBackendComparisonHonorsConfiguredLimitForExternalRows|LocomoBackendComparisonDuplicateExternalRowsDoNotExpandTopK)' -count=1` proves `--limit 1` reaches external adapter scoring and duplicate external rows cannot expand the top-K window.
  - Result: reproducible backend comparison reports now use the operator-requested top-K window consistently across local and external backends.

- 2026-05-22: LOCOMO external adapter scoring now clamps top-K rows and rejects out-of-conversation stable IDs.
  - Evidence target: `go test ./cmd/goncho-bench -run 'TestLocomoBackendComparison(LimitsExternalRowsToTopK|RejectsExternalOutOfConversationMemoryID)' -count=1` proves comparable external rows obey the requested top-K window and cannot return a stable `memory_id` from a different `conversation_id` than the question.
  - Result: the Go scorer now enforces the documented conversation-scoped backend comparison contract before stable-ID scoring, so external adapters cannot get comparable credit by over-returning rows or crossing conversation boundaries.

- 2026-05-22: benchmark docs now surface conversation-scoped backend comparison.
  - Evidence target: `go test . -run TestBenchmarkDocsMentionConversationScopedBackendComparison -count=1` proves README, Retrieval Benchmarks, operator runbook, and external adapter docs say LOCOMO backend comparison is conversation-scoped.
  - Result: public benchmark methodology now explains why duplicate or near-duplicate content in another conversation cannot win by content-only matching before stable-ID scoring.

- 2026-05-22: public release metadata smoke now checks documented latest metadata.
  - Evidence target: `go test . -run 'Test(PublicReleaseSmokeChecksDocumentedLatestMetadata|PublicDocsExplainDocumentedLatestPublicReleaseSmoke)' -count=1` proves `make public-release-smoke` checks the documented public `@latest` version and published date, and first-touch public docs explain that guard.
  - Result: ecosystem-readiness smoke now catches drift between official public module metadata and the documented v0.1.0 / May 20, 2026 milestone instead of accepting any `Version`/`Time` fields.

- 2026-05-22: first-touch public docs now surface the public docs site smoke.
  - Evidence target: `go test . -run 'Test(DocsSiteSmokeBuildsPublicDocs|PublicDocsMentionDocsSiteSmoke)' -count=1` proves `make docs-site-smoke` checks the local docs-site build with `npm run build`, and first-touch public docs mention the command.
  - Result: ecosystem-readiness docs now expose a narrow proof for the public docs site signal without claiming local smoke proves remote hosting or indexing.

- 2026-05-22: first-touch public docs now surface the package documentation smoke.
  - Evidence target: `go test . -run 'Test(PackageDocSmokeChecksLocalGoDoc|PublicDocsMentionPackageDocSmoke)' -count=1` proves `make package-doc-smoke` checks local package docs with `go doc .`, and first-touch public docs mention the command.
  - Result: ecosystem-readiness docs now expose a narrow proof for the package documentation signal without claiming that local smoke proves remote pkg.go.dev indexing.

- 2026-05-22: first-touch public docs now surface the local go.mod metadata smoke.
  - Evidence target: `go test . -run 'Test(LocalModuleSmokeChecksGoModMetadata|PublicDocsMentionLocalModuleSmoke)' -count=1` proves `make local-module-smoke` checks the local module path and Go version with `go list -m -json`, and first-touch public docs mention the command.
  - Result: ecosystem-readiness docs now expose a narrow proof for the valid Go module signal without conflating it with public `@latest` metadata or root CLI installability.

- 2026-05-22: first-touch public docs now surface the external backend comparison smoke.
  - Evidence target: `go test . -run TestPublicDocsMentionBackendComparisonSmoke -count=1` proves README, docs home, current-capabilities, and quick-start docs mention `make bench-locomo-backends-smoke`.
  - Result: ecosystem-readiness docs now expose the CI-safe local proof command for external adapter comparison without rerunning or changing frozen benchmark artifacts.

- 2026-05-22: first-touch public docs now surface the external adapter contract.
  - Evidence target: `go test . -run TestPublicDocsSurfaceExternalAdapterContract -count=1` proves README, docs home, current-capabilities, and quick-start docs mention the external adapter contract and current agentmemory PR #583 stable-ID status.
  - Result: ecosystem-readiness docs now expose adapter/upstream credibility at adoption time without overstating backend scores or root CLI installability.

- 2026-05-22: first-touch public docs now link benchmark methodology.
  - Evidence target: `go test . -run TestPublicDocsLinkRetrievalBenchmarksReference -count=1` proves README, docs home, current-capabilities, and quick-start docs link the Retrieval Benchmarks reference.
  - Result: public package adoption now exposes deterministic benchmark methodology and stable-ID backend comparison evidence without making benchmark claims in setup prose.

- 2026-05-22: public docs now warn against root-level `go install` overclaims.
  - Evidence target: `go test . -run TestPublicDocsWarnRootGoInstallIsUnsupported -count=1` proves README, docs home, current-capabilities, and quick-start docs say the root module is not a root `go install` target.
  - Result: first-touch docs preserve the `go get github.com/TrebuchetDynamics/goncho@latest` library path without implying an unavailable root CLI install.

- 2026-05-22: public docs now surface the v0.1.0 published date.
  - Evidence target: `go test . -run TestPublicDocsMentionPublishedReleaseDate -count=1` proves README, docs home, current-capabilities, and quick-start docs mention `published May 20, 2026`.
  - Result: first-touch docs now show both public version and published-date signals from the official module metadata without implying a newer tag.

- 2026-05-22: public adoption docs now use version-qualified `go get`.
  - Evidence target: `go test . -run TestPublicDocsUseLatestQualifiedGoGet -count=1` proves README, docs home, current-capabilities, and quick-start docs mention `go get github.com/TrebuchetDynamics/goncho@latest`.
  - Result: first-touch setup guidance matches the public `@latest` release signal while keeping the root module framed as a library package.

- 2026-05-22: public release metadata smoke added.
  - Evidence target: `make public-release-smoke` checks `go list -m -json github.com/TrebuchetDynamics/goncho@latest` for public version and published-time metadata.
  - Result: the pkg.go.dev-style `Version` and `Published` signal is locally checkable before broader ecosystem smoke and release decisions.

- 2026-05-22: docs home now frames the root module as a library package.
  - Evidence target: `go test . -run TestPublicDocsFrameRootModuleAsLibrary -count=1` proves README, docs home, current-capabilities, and quick-start docs say the root module is a library package.
  - Result: first-touch public docs preserve `go get` library semantics and avoid implying root-level CLI installability.

- 2026-05-22: docs home now names the current public `@latest` release as v0.1.0.
  - Evidence target: `go test . -run TestPublicDocsMentionLatestReleaseVersion -count=1` proves README, docs home, current-capabilities, and quick-start docs mention v0.1.0.
  - Result: first-touch public docs show the official tagged release signal without implying checkout-local benchmark CLI availability at `@latest`.

- 2026-05-22: README and docs home now expose the narrower public module smoke.
  - Evidence target: `go test . -run TestPublicAdoptionDocsMentionPublicModuleSmoke -count=1` proves README, docs home, current-capabilities, and quick-start docs mention `make public-module-smoke`.
  - Result: public adoption docs separate the broad ecosystem smoke from the external-import-only proof for `github.com/TrebuchetDynamics/goncho@latest`.

- 2026-05-22: docs home now surfaces local ecosystem smoke.
  - Evidence target: `go test . -run TestPublicDocsMentionEcosystemSmoke -count=1` proves README, docs home, operator runbook, current-capabilities, and quick-start docs mention `make ecosystem-smoke`.
  - Result: public adoption docs expose the local proof command for module resolution, package docs, external importability, and checkout-local benchmark CLI readiness.

- 2026-05-22: docs home and quick-start docs now link the public Go reference.
  - Evidence target: `go test . -run TestPublicDocsLinkGoReference -count=1` proves README, docs home, current-capabilities, and quick-start docs link `https://pkg.go.dev/github.com/TrebuchetDynamics/goncho`.
  - Result: public adoption docs surface pkg.go.dev API reference at first use instead of hiding it in status pages.

- 2026-05-21: operator-facing release smoke docs now mention the release metadata guard.
  - Evidence target: `go test . -run TestReleaseSmokeDocsMentionMetadataGuard -count=1` proves README, quick-start, and runbook release-smoke guidance mention release metadata checks.
  - Result: public docs stay aligned with the local pre-tag gate instead of describing only ecosystem smoke plus Go/docs checks.

- 2026-05-22: release metadata now has an explicit smoke target.
  - Evidence target: `make release-metadata-smoke` runs tag/changelog consistency and release-smoke docs drift tests before broader release checks.
  - Result: operators can check changelog/tag consistency and release-smoke docs directly, and `make release-smoke` includes that guard before ecosystem validation.

- 2026-05-22: changelog release headings are now guarded against untagged version overclaims.
  - Evidence target: `go test . -run TestChangelogReleaseHeadingsHaveMatchingTags -count=1` proves each `## vX.Y.Z - ...` changelog release heading has a matching local git tag.
  - Result: public release notes can keep candidate notes without implying that untagged versions are already published.

- 2026-05-22: blank `goncho_review` list status values now default to open review items.
  - Evidence target: `go test . -run TestReviewToolTreatsBlankStatusAsOpenDefault -count=1` proves whitespace-only `status` behaves like omitted `status` and does not leak resolved items into the default review queue.
  - Result: review queue inspection is safer when host/tool callers pass blank form values instead of omitting optional fields.

- 2026-05-22: invalid `goncho_review` resolve resolution values now return enum-specific guidance.
  - Evidence target: `go test . -run TestReviewToolRejectsInvalidResolveResolution -count=1` proves an invalid `resolution` value is rejected without closing the open review item.
  - Result: lifecycle review queues are safer when host/tool callers bypass schema enum validation.

- 2026-05-22: same-timestamp review item ID collision fixed.
  - Evidence target: `go test . -run TestCreateReviewItemAllowsDistinctItemsWithSameCreatedAt -count=1` proves two distinct review items sharing one `CreatedAt` get distinct IDs and remain listable.
  - Result: review queues are safer when lifecycle scanners create multiple findings in the same timestamp bucket.

- 2026-05-22: `goncho_review` list filter validation added.
  - Evidence target: `go test . -run TestReviewToolRejectsInvalidListFilters -count=1` proves invalid `status` and `kind` list filters return operator-visible errors instead of empty review queues.
  - Result: review queue inspection is safer when host/tool callers bypass schema enum validation.

- 2026-05-22: local release smoke added.
  - Evidence target: `make release-smoke` runs `make ecosystem-smoke`, `go test ./...`, `go vet ./...`, `go test -race ./...`, and the docs-site build.
  - Result: next v0.1.x prep has one local pre-tag command without claiming CI or creating a tag.

- 2026-05-22: `goncho_review` review-chain filters added.
  - Evidence target: `go test . -run TestReviewToolFiltersReviewChainsBySubjectAndRelatedID -count=1` proves `subject_id` plus `related_id` narrows open review items to one matching chain edge.
  - Result: review/staleness/supersession items are easier to inspect without losing historical evidence.

- 2026-05-22: ecosystem smoke added for core public release-readiness signals.
  - Evidence target: `make ecosystem-smoke` runs public module resolution, local `go doc .`, external import smoke, and checkout-local benchmark CLI installation.
  - Result: the milestone now has one operator command for library importability plus local benchmark CLI readiness without overstating `cmd/goncho-bench@latest`.

- 2026-05-22: public module adoption smoke added for `github.com/TrebuchetDynamics/goncho@latest`.
  - Evidence target: `make public-module-smoke` creates a temporary external Go module, runs `go get github.com/TrebuchetDynamics/goncho@latest`, and compiles a minimal public API import.
  - Result: release readiness now separates library importability proof from the still-checkout-local benchmark CLI.

- 2026-05-22: public `@latest` still resolves to v0.1.0, so `go install github.com/TrebuchetDynamics/goncho/cmd/goncho-bench@latest` does not work yet.
  - Evidence: public install reports module `github.com/TrebuchetDynamics/goncho@latest` found at `v0.1.0`, but it does not contain `cmd/goncho-bench`.
  - Result: docs now point benchmark CLI users at checkout-local `make install-smoke` / `go install ./cmd/goncho-bench` until the next v0.1.x tag contains the command.

- 2026-05-22: generated primer/token-budget E2E coverage added for the public `goncho_context` tool.
  - Focused evidence: `go test . -run TestGonchoGoalPublicContextToolGeneratesPrimerWithinTokenBudgetE2E -count=1` passed.
  - Result: public context-tool coverage now proves generated orientation output preserves the newest in-budget turns and excludes older turns outside `max_tokens`.

- 2026-05-21 20:11 CST: stale `cmd/goncho-bench` expectation-drift blocker from 2026-05-20 is resolved on current `main`.
  - Focused evidence: `go test ./cmd/goncho-bench -run 'TestClassifyFailureCasesSelectsHardRanksAndCategories|TestWriteFailureCategoryReportsEmitsJSONLAndMarkdown' -count=1` passed.
  - Full Go evidence: `go test ./... -count=1` passed.
  - Result: benchmark classifier expectation drift no longer blocks Go verification.

- 2026-05-19: stale full-verification blocker resolved. Current release gate passes with:
  - `go test ./integration/gormes`
  - `go test ./...`
  - `cd docs-site && npm run build`

## Next roadmap items

- Continue lifecycle trust work: temporal validity, supersession chains, and confidence/freshness scoring.
- Expand graph/cognitive-map features behind deterministic tests.
- Add optional PostgreSQL/team adapter only after local SQLite API remains stable.

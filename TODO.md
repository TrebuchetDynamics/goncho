# Goncho TODO

## Release state

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

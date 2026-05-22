# Changelog

## Unreleased

### Changed

- Updated public `@latest` release metadata docs and smoke guards after v0.1.1 tag publication.

## v0.1.1 - 2026-05-22

### Added

- Added `make docs-site-smoke` to verify the public documentation site builds locally with `npm run build`.
- Added `make package-doc-smoke` to verify checkout-local package documentation renders through `go doc .`.
- Added `make local-module-smoke` to verify checkout-local go.mod module path and Go version metadata from `go list -m -json`.
- Added `make public-release-smoke` to verify public `@latest` version and published-time metadata from `go list -m -json`.
- Added `make release-metadata-smoke` as an explicit guard for changelog release headings, local git tags, and release-smoke docs.
- Added `make release-smoke` as a local pre-tag gate for release metadata, ecosystem smoke, Go tests, vet, race tests, and docs build.
- Added effective `status` echoes to `goncho_review` list responses so operators can audit silent default-open review queue requests.
- Added `workspace_id` filters to `goncho_review` list requests so operators can inspect workspace-specific review queues.
- Added `subject_id` and `related_id` filters to `goncho_review` list requests so operators can inspect review and supersession chains.
- Added `make ecosystem-smoke` to run the core public module, package-doc, external-import, and checkout-local benchmark CLI readiness checks together.
- Added `make public-module-smoke` to verify `github.com/TrebuchetDynamics/goncho@latest` imports and compiles from a fresh external Go module.
- Added `make install-smoke` to verify the checkout-local `cmd/goncho-bench` install path without touching a developer's normal `GOBIN`.

### Changed

- Checked-in LOCOMO smoke benchmark artifacts now include `goncho-no-rank` retrieval and backend-comparison baselines.
- LOCOMO backend-comparison reports now include a `goncho-no-rank` no-ranking baseline alongside current Goncho.
- LOCOMO retrieval reports now include a `goncho-no-rank` no-ranking baseline alongside current Goncho.
- LOCOMO retrieval and backend-comparison markdown now include one-command reproduction lines.
- LOCOMO retrieval markdown now includes converted-artifact checksums when metadata is available.
- LOCOMO backend-comparison markdown now includes dataset source, checksum, converted-artifact checksum, and license provenance when metadata is available.
- LOCOMO backend-comparison reports now include per-backend category metrics.
- LOCOMO backend-comparison markdown now includes per-backend failure-category counts.
- LOCOMO backend-comparison reports now include per-backend latency distribution stats.
- LOCOMO backend-comparison markdown now includes per-backend insert latency and RSS metrics.
- LOCOMO backend-comparison reports now record per-backend NDCG@5 and NDCG@10 metrics in JSON and markdown artifacts.
- LOCOMO backend-comparison reports now record LOCOMO leakage checks in JSON and markdown artifacts.
- LOCOMO retrieval and backend-comparison reports now record converted fixture database byte sizes in JSON and markdown artifacts.
- LOCOMO retrieval reports now record per-system NDCG@5 and NDCG@10 metrics in JSON and markdown artifacts.
- LOCOMO retrieval reports now record per-system latency distribution stats in JSON and markdown artifacts.
- LOCOMO retrieval reports now record per-system failure-category counts in JSON and markdown artifacts.
- LOCOMO retrieval and backend-comparison reports now record deterministic memory token estimates in JSON and markdown artifacts.
- LOCOMO retrieval reports now record per-system search latency and RSS metrics in JSON and markdown artifacts.
- LOCOMO retrieval and backend-comparison reports now record the effective top-K scoring window in JSON and markdown artifacts.
- LOCOMO failure-audit miss notes now report the actual retrieved top-K window instead of hard-coding top 10.
- LOCOMO failure audits now reject out-of-conversation gold stable IDs before writing failure rows.
- LOCOMO failure audits now reject unknown gold stable IDs before writing failure rows.
- LOCOMO failure audits now reject question conversation mismatches before writing failure rows.
- LOCOMO failure audits now reject unknown question IDs before writing failure rows.
- LOCOMO failure audits now reject out-of-conversation retrieved stable IDs before writing top-hit rows.
- LOCOMO backend-comparison failure audits now reject unknown retrieved stable IDs instead of emitting blank top-hit rows.
- LOCOMO SQLite FTS retrieval now skips temporary database setup for tokenless queries and uses the existing recency fallback directly.
- LOCOMO failure audits now reject unknown retrieved stable IDs instead of emitting blank top-hit rows.
- LOCOMO leakage checks now reuse the precomputed conversation index instead of rebuilding it per report.
- LOCOMO direct retrieval helpers now return no IDs for non-positive limits before invoking local backends.
- LOCOMO Goncho adapters now cap stable-ID fan-out to the requested top-K window when duplicate content maps to multiple stable IDs.
- LOCOMO fixture loading now rejects duplicate `gold_memory_ids` within a question before stable-ID scoring.
- LOCOMO fixture loading now rejects unknown or out-of-conversation `gold_memory_ids` before stable-ID scoring.
- LOCOMO fixture loading now rejects duplicate `memory_id` and `question_id` values before stable-ID scoring.
- LOCOMO smoke/full retrieval reports now honor the configured retrieval limit instead of always evaluating every local system with top 10.
- LOCOMO backend comparison now honors the configured retrieval limit when scoring every backend, including external adapter rows, and duplicate external rows no longer expand the top-K window.
- LOCOMO external backend comparison now clamps comparable external rows to the requested top-K window and rejects stable memory IDs from a different conversation than the question.
- `make public-release-smoke` now checks the documented public `@latest` version and published date, not just the presence of release metadata fields.
- `goncho_review` list requests now treat blank `status` values like omitted status and default to open review items.
- `goncho_review` resolve responses now include workspace IDs.
- `goncho_review` resolve responses now include original creation timestamps.
- `goncho_review` resolve responses now include original review reasons.
- `goncho_review` resolve responses now include evidence IDs.
- `goncho_review` resolve responses now include peer/session scope.
- `goncho_review` resolve responses now include review kind.
- `goncho_review` resolve responses now include review-chain identifiers.
- `goncho_review` resolve responses now include `resolved_at` audit timestamps.
- `goncho_review` resolve requests now return enum-specific guidance for invalid `resolution` values.
- Review item IDs now include a deterministic field fingerprint so distinct same-timestamp review items do not collide.
- `goncho_review` list requests now reject invalid `status` and `kind` filters instead of silently returning an empty queue.

### Documentation

- Clarified benchmark docs that LOCOMO backend comparison is conversation-scoped before stable-ID scoring.
- Clarified first-touch public docs that public release metadata smoke checks the documented public `@latest` version and published date.
- Surfaced the public docs site smoke command from first-touch public docs.
- Surfaced the package documentation smoke command from first-touch public docs.
- Surfaced the local go.mod metadata smoke command from first-touch public docs.
- Surfaced the external backend comparison smoke command from first-touch public docs.
- Surfaced the external adapter contract and agentmemory PR #583 stable-ID status from first-touch public docs.
- Linked first-touch public docs to the Retrieval Benchmarks methodology and stable-ID backend comparison reference.
- Clarified first-touch public docs that the root module is not a root `go install` target.
- Surfaced the public v0.1.0 published date in first-touch package docs.
- Clarified first-touch public adoption docs to use `go get github.com/TrebuchetDynamics/goncho@latest` for the library package.
- Surfaced public release metadata smoke guidance and guarded first-touch docs for version/published-time proof.
- Surfaced docs-home root-library framing and guarded first-touch public install semantics.
- Surfaced docs-home `v0.1.0` public latest status and guarded first-touch release-version docs.
- Surfaced README and docs-home guidance for the narrower public module smoke and guarded public adoption docs.
- Surfaced ecosystem smoke from docs home and guarded public ecosystem-smoke mentions in release metadata checks.
- Linked the public Go reference from docs home and quick-start docs, guarded by release metadata smoke checks.
- Clarified README, quick-start, and operator runbook release-smoke guidance so it includes the release metadata guard.
- Added a release-metadata guard so changelog release headings must match local git tags.
- Corrected benchmark CLI install guidance so public docs no longer claim `cmd/goncho-bench@latest` before a tag contains the command.
- Documented public `goncho_context` generated-primer token-budget E2E coverage in the README and current-capabilities docs.
- Added root package documentation so pkg.go.dev exposes Goncho's public memory-kernel purpose and evidence-before-belief rule.
- Added README package-status framing with public module verification and benchmark-methodology signals.
- Added public package/status framing to the current-capabilities docs, including pkg.go.dev, v0.1.x, benchmark evidence, and stable-ID backend comparison signals.
- Clarified library install guidance versus the installable `goncho-bench` benchmark CLI in the README and quick-start docs.
- Clarified current benchmark roadmap and backlog status after the LOCOMO stable-ID backend comparison freeze and stale benchmark blocker resolution.

### Benchmark candidate milestone from 2026-05-20

#### Added

- `cmd/goncho-bench`, a local LongMemEval-style retrieval benchmark runner that reports `R@5`, `R@10`, and `MRR` from JSONL memory/question fixtures.
- `--runs` loop mode for repeated deterministic retrieval benchmark runs.
- Explicit `recall_any_at_5` and `recall_any_at_10` fields for LongMemEval retrieval-table comparisons.
- Scientific benchmark metadata: dataset revision, dataset SHA256, Go/runtime environment, leakage counts, and failure-audit JSONL.
- Deterministic baselines for `random`, `bm25`, `sqlite-fts5`, `goncho-no-rank`, and `goncho`.
- `make bench-longmemeval-s-smoke` and `make bench-longmemeval-s` clean-room benchmark targets.
- BM25-style lexical conclusion ranking so search results are ordered by query/content token relevance before recency tie-breaks.
- Benchmark roadmap covering LOCOMO, InfiniteBench, RULER, BABILong, BEIR, and real-world agent replay as future scientific evaluations.
- Retrieval benchmark documentation and a tiny deterministic fixture for harness validation.
- Frozen LOCOMO candidate-generation milestone: Goncho recall_any@5 `0.5247 -> 0.6014`, recall_any@10 `0.5873 -> 0.6791`, MRR `0.4104 -> 0.4690`, and BM25-win `missing_candidate` failures `164 -> 2` without LLM judgment, answer scoring, gold-ID hacks, or ranking changes.
- Stable-ID LOCOMO external-backend adapter comparison harness for Goncho, BM25, SQLite FTS5, agentmemory, and mem0, with centralized ID scoring and not-comparable reporting.

#### Changed

- Widened LOCOMO lexical candidate generation while preserving LongMemEval-S benchmark performance.
- Deepened retrieval architecture by moving search/context orchestration behind an internal retrieval module without public API changes.

#### Documentation

- Added benchmark operator guidance, backend-adapter rules, LOCOMO milestone notes, and architecture design/implementation plans for retrieval and lifecycle module deepening.

## v0.1.0 - 2026-05-19

Initial tagged release for local-first Goncho agent memory.

### Added

- Embedded Go service API for profiles, search, context, chat, conclusions, review, handoff, and local memory tools.
- SQLite-backed local storage and migrations.
- Honcho-compatible primitives and public Goncho tools:
  - `goncho_context`
  - `goncho_search`
  - `goncho_remember`
  - `goncho_review`
  - `goncho_handoff`
- MCP-style memory tool contracts:
  - `store_memory`
  - `retrieve_memory`
  - `update_memory`
  - `summarize_memories`
  - `forget_memory`
- Local deterministic E2E coverage for service lifecycle, HTTP lifecycle, public tool restart persistence, prompt-injection quarantine, stale code-claim verification, and negative drift anchors.
- Prompt-injection-like import quarantine that preserves suspicious imported content as skipped evidence while excluding it from trusted context/search.
- Live code-claim verification for remembered file/path claims against a local repository root.
- Negative drift-anchor detector for known failed paths and dead-end memory.
- `integration/gormes` adapter package for Gormes-style Go agent hosts.
- Operator runbook and Gormes integration documentation.

### Release validation

Validated before tagging:

```sh
go test ./integration/gormes
go test ./...
cd docs-site && npm run build
```

### Compatibility notes

- Module path: `github.com/TrebuchetDynamics/goncho`.
- The base workflow is local-first and SQLite-backed.
- Goncho is pre-1.0; public APIs may evolve, but v0.1.0 establishes the initial importable service and Gormes adapter surface.

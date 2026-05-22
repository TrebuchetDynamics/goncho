# Changelog

## Unreleased

### Added

- Added `make release-smoke` as a local pre-tag gate for ecosystem smoke, Go tests, vet, race tests, and docs build.
- Added `subject_id` and `related_id` filters to `goncho_review` list requests so operators can inspect review and supersession chains.
- Added `make ecosystem-smoke` to run the core public module, package-doc, external-import, and checkout-local benchmark CLI readiness checks together.
- Added `make public-module-smoke` to verify `github.com/TrebuchetDynamics/goncho@latest` imports and compiles from a fresh external Go module.
- Added `make install-smoke` to verify the checkout-local `cmd/goncho-bench` install path without touching a developer's normal `GOBIN`.

### Changed

- `goncho_review` list requests now treat blank `status` values like omitted status and default to open review items.
- `goncho_review` resolve requests now return enum-specific guidance for invalid `resolution` values.
- Review item IDs now include a deterministic field fingerprint so distinct same-timestamp review items do not collide.
- `goncho_review` list requests now reject invalid `status` and `kind` filters instead of silently returning an empty queue.

### Documentation

- Added a release-metadata guard so changelog release headings must match local git tags.
- Corrected benchmark CLI install guidance so public docs no longer claim `cmd/goncho-bench@latest` before a tag contains the command.
- Documented public `goncho_context` generated-primer token-budget E2E coverage in the README and current-capabilities docs.
- Added root package documentation so pkg.go.dev exposes Goncho's public memory-kernel purpose and evidence-before-belief rule.
- Added README package-status framing with public module verification and benchmark-methodology signals.
- Added public package/status framing to the current-capabilities docs, including pkg.go.dev, v0.1.x, benchmark evidence, and stable-ID backend comparison signals.
- Clarified library install guidance versus the installable `goncho-bench` benchmark CLI in the README and quick-start docs.
- Clarified current benchmark roadmap and backlog status after the LOCOMO stable-ID backend comparison freeze and stale benchmark blocker resolution.

## v0.1.1 candidate notes - 2026-05-20

### Added

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

### Changed

- Widened LOCOMO lexical candidate generation while preserving LongMemEval-S benchmark performance.
- Deepened retrieval architecture by moving search/context orchestration behind an internal retrieval module without public API changes.

### Documentation

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

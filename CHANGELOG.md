# Changelog

## Unreleased

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

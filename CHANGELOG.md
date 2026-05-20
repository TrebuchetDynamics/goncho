# Changelog

## Unreleased

### Added

- `cmd/goncho-bench`, a local LongMemEval-style retrieval benchmark runner that reports `R@5`, `R@10`, and `MRR` from JSONL memory/question fixtures.
- `--runs` loop mode for repeated deterministic retrieval benchmark runs.
- Lexical conclusion ranking so search results are ordered by query/content token overlap before recency tie-breaks.
- Retrieval benchmark documentation and a tiny deterministic fixture for harness validation.

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

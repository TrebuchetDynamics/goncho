---
title: Current Capabilities
description: What Goncho supports today and what remains architecture direction.
---

Goncho is usable as a pre-1.0 Go library, with v0.1.x focused on the importable local memory kernel, compatibility surfaces, and deterministic local E2E proof.

## Public Package Surface

Current public package signals:

- Module path: `github.com/TrebuchetDynamics/goncho`.
- API reference: [pkg.go.dev/github.com/TrebuchetDynamics/goncho](https://pkg.go.dev/github.com/TrebuchetDynamics/goncho).
- Tagged release stream: public `@latest` currently resolves to v0.1.1, published May 22, 2026; v0.1.x is the active release line for the importable service, Gormes adapter surface, and benchmark CLI.
- Local release smoke: `make release-smoke` runs release metadata checks, ecosystem smoke, Go tests, vet, race tests, and the docs-site build before local pre-tag decisions.
- Ecosystem smoke: `make ecosystem-smoke` verifies public release metadata, local Go module metadata, local package documentation, public docs site build, a temporary external Go module import, and checkout-local benchmark CLI installation.
- Public release metadata smoke: `make public-release-smoke` checks the documented public `@latest` version and published date from `go list -m -json github.com/TrebuchetDynamics/goncho@latest`.
- Local module metadata smoke: `make local-module-smoke` checks the checkout `go.mod` module path and Go version from `go list -m -json`.
- Package documentation smoke: `make package-doc-smoke` checks that local package docs render through `go doc .`.
- Public docs site smoke: `make docs-site-smoke` checks the local documentation site build with `npm run build`.
- Public import smoke: `make public-module-smoke` creates a temporary external Go module, runs `go get github.com/TrebuchetDynamics/goncho@latest`, and compiles a minimal import of the public service API.
- Installable command source: `./cmd/goncho-bench`; `go install github.com/TrebuchetDynamics/goncho/cmd/goncho-bench@latest` installs the public benchmark CLI, and `make install-smoke` verifies the checkout-local path.
- Public tool proof: `goncho_context` has E2E coverage for generated primer behavior under `max_tokens`, preserving newest in-budget turns and excluding older out-of-budget turns.
- Benchmark evidence: LongMemEval-S and LOCOMO reports use deterministic ID scoring, leakage checks, failure audits, reproducible smoke targets, and stable-ID backend comparison artifacts; see [Retrieval Benchmarks](/reference/retrieval-benchmarks/) for methodology, the external adapter contract, current agentmemory PR #583 stable-ID status, and the CI-safe `make bench-locomo-backends-smoke` proof.

The root module is a library package, not a CLI binary and not a root `go install` target. Treat the public package as an ecosystem component with reproducible evidence, while still treating deeper graph, lifecycle, and team-memory features as roadmap direction until their APIs are explicit.

| Capability | Today | Direction |
| --- | --- | --- |
| SQLite persistence | `memory.OpenSqlite` initializes service tables; `RunMigrations` initializes observation and audit tables | Clearer operational migration docs and lifecycle guidance |
| Peer cards | `SetProfile`, `Profile`, directional peer-card support | Richer belief provenance |
| Search | `Search` over stored conclusions and fallback session turns | Stronger evidence lineage and ranking diagnostics |
| Recall trace | `Recall` exposes the scored `RecallTrace` with candidates, provenance, selected/rejected memories, and warnings before projection | More host-facing replay and review tooling around trace evidence |
| Context assembly | `Context` returns peer card, conclusions, summaries, search hits, and recent turns | Compact orientation packs with stronger provenance |
| Session summaries | Deterministic short and long summary slots | Hook-native consolidation around lifecycle boundaries |
| Local markdown memory | Shipped local markdown memory store and memory tools | Review and repair workflows around editable memory |
| Honcho compatibility | Honcho-style names and compatibility harnesses exist | Clearer migration boundaries |
| Confidence and staleness | Conceptual direction | First-class scoring contract |
| Temporal validity | Conceptual direction | Scoped validity windows |
| Negative memory review | Conceptual direction | First-class review and repair products |

:::note[Current status]
This site documents both the shipped library and the architecture direction. Pages call out conceptual examples when they describe constraints Goncho has not exposed as stable API.
:::

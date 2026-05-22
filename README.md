# Goncho

**High-trust local memory for Go-native AI agent runtimes.**

Goncho is the memory kernel for agent hosts that need durable local state, auditability, scoped recall, review warnings, and live-verification discipline without a Python service, Docker sidecar, hosted memory API, vector database wrapper, or giant tool catalog.

It is designed for the Trebuchet local-first agent stack:

- **Gormes** — a Go-native AI agent runtime for Linux, Windows, macOS, and Termux on Android. One binary runs providers, tools, skills, sessions, local memory, chat, and gateways with no Python or Docker required.
- **Navivox** — an Android app in development that turns a phone into an AI agent server with local memory, chat, and gateways.
- **Goncho** — the high-trust memory layer underneath: evidence, claims, scoped temporal beliefs, context packs, review queues, and live verification.

Goncho's rule is simple:

```text
evidence before belief
live verification before action
```

If memory says a file exists, verify it. If memory says a migration was approved, verify it. If memory says an API path still exists, verify it. Goncho treats memory as orientation until current evidence says it is safe to act.

Use Goncho as an embedded Go module:

```bash
go get github.com/TrebuchetDynamics/goncho@latest
```

From a checkout, verify the reproducible benchmark CLI builds and starts when you need local retrieval reports:

```bash
make install-smoke
```

The root module is a library package, not a root `go install` target; `goncho-bench` is the installable command in `./cmd/goncho-bench`. Public `@latest` currently resolves to v0.1.1, published May 22, 2026, and includes the benchmark CLI.

[![Go Reference](https://pkg.go.dev/badge/github.com/TrebuchetDynamics/goncho.svg)](https://pkg.go.dev/github.com/TrebuchetDynamics/goncho)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Package Status

Goncho is pre-1.0, but it has the public release signals needed to evaluate it as an ecosystem component: a tagged v0.1.1 release published May 22, 2026, a valid Go module, pkg.go.dev API docs, public docs, reproducible benchmark commands, deterministic benchmark methodology, and stable-ID backend comparison artifacts. The LOCOMO backend comparison is conversation-scoped so duplicate content in other conversations cannot win by content alone. Benchmark methodology, the external adapter contract, and current agentmemory PR #583 stable-ID status live in [Retrieval Benchmarks](docs-site/src/content/docs/reference/retrieval-benchmarks.md).

Verify public release metadata, local Go module metadata, package documentation, public docs site build, external importability, and the checkout-local benchmark CLI without editing another project:

```bash
make ecosystem-smoke
```

For the narrower public release metadata proof only, run `make public-release-smoke`; it checks the documented public `@latest` version and published date from `go list -m -json`. For the narrower local go.mod metadata proof only, run `make local-module-smoke`; it checks the module path and Go version from `go list -m -json`. For the narrower package documentation proof only, run `make package-doc-smoke`; it checks that local package docs render through `go doc .`. For the narrower public docs site proof only, run `make docs-site-smoke`; it checks the local docs-site build with `npm run build`. For the narrower external import proof only, run `make public-module-smoke`. For the CI-safe external backend comparison proof, run `make bench-locomo-backends-smoke`.

Use `go get github.com/TrebuchetDynamics/goncho@latest` to depend on the library package. For the command-line benchmark runner, use `go install github.com/TrebuchetDynamics/goncho/cmd/goncho-bench@latest` or checkout-local `make install-smoke`.

---

## Why Goncho

Most agent memory systems optimize for breadth: more connectors, more tools, more autonomous behavior. Goncho optimizes for trust: memory correctness, bounded writes, reproducible retrieval, local state, and verification before action.

Goncho exists because Gormes and Navivox need memory that can run anywhere the agent runs: a workstation, small server, Windows laptop, WSL2 shell, macOS terminal, or Android phone through Termux. The memory layer cannot assume Python packaging, Docker, Redis, hosted vector infrastructure, or always-online cloud services.

Goncho is inspired by broad integration systems like [`agentmemory`](docs/opensource-memory-systems/agentmemory/README.md), but it makes a different product bet:

```text
agentmemory: broad integration layer
Gormes:      Go-native agent runtime
Navivox:     Android agent server
Goncho:      high-trust local memory kernel
```

The core abstraction is not “top-k chunks in a prompt.” It is:

```text
raw evidence
  -> claims
  -> scoped temporal beliefs
  -> task-specific orientation
  -> agent action
  -> review, verification, revision, or forgetting
```

Vectors and search are useful. They are not the source of truth. Retrieval can suggest; verification decides.

---

## Where Goncho Fits

```text
Navivox Android app
  -> Gormes Go runtime
    -> Goncho local memory
    -> chat gateways
    -> providers, tools, skills
```

Gormes owns the agent runtime: provider turns, tools, skills, profiles, sessions, TUI, dashboard, and chat gateways. Goncho owns memory integrity: what was observed, what was concluded, which profile can read it, whether it may be stale, and what must be verified before action.

Navivox brings that stack onto Android. The phone becomes a local agent server: chat interface, gateway hub, and memory-bearing runtime. Goncho's job in that environment is to keep memory useful without requiring a heavyweight server deployment.

The boundary is intentional:

| Layer | Responsibility |
| --- | --- |
| Navivox | Android app, mobile UX, phone-hosted agent server, local chat and gateway controls. |
| Gormes | Go-native agent runtime, providers, tools, skills, profiles, sessions, TUI/dashboard, gateways. |
| Goncho | Local memory kernel, scoped recall, evidence capture, review warnings, stale-claim verification, handoffs. |

---

## What Goncho Provides

| Capability | Status |
| --- | --- |
| Embedded Go service | Implemented |
| SQLite local storage | Implemented |
| Profile, search, context, chat, conclude APIs | Implemented |
| Honcho-compatible primitives | Implemented |
| MCP-style memory tools | Implemented |
| Public tools: context, search, remember, review, handoff | Implemented |
| Multi-profile memory isolation | Implemented |
| Gormes profile directories | Implemented |
| Prompt-injection quarantine | Implemented |
| Stale code-claim verification | Implemented |
| Negative drift anchors | Implemented |
| Review queues | Experimental |
| Graph/cognitive-map layer | Planned |
| PostgreSQL team adapter | Planned |

---

## Quick Start

```go
package main

import (
    "context"
    "fmt"

    "github.com/TrebuchetDynamics/goncho"
    "github.com/TrebuchetDynamics/goncho/memory"
)

func main() {
    ctx := context.Background()

    store, err := memory.OpenSqlite("memory.db", 0, nil)
    if err != nil {
        panic(err)
    }
    defer func() { _ = store.Close(ctx) }()

    if err := goncho.RunMigrations(store.DB()); err != nil {
        panic(err)
    }

    svc := goncho.NewService(store.DB(), goncho.Config{
        WorkspaceID:    "my-agent",
        ObserverPeerID: "assistant",
    }, nil)

    _, err = svc.Conclude(ctx, goncho.ConcludeParams{
        ProfileID:  "mineru",
        Peer:       "telegram:12345",
        Conclusion: "User prefers SQLite over hosted vector services.",
    })
    if err != nil {
        panic(err)
    }

    pack, err := svc.Context(ctx, goncho.ContextParams{
        ProfileID: "mineru",
        Peer:      "telegram:12345",
        Query:     "database preferences",
        MaxTokens: 2000,
    })
    if err != nil {
        panic(err)
    }

    fmt.Println(pack.Representation)
}
```

---

## Core API

Common embedded calls:

```go
svc := goncho.NewService(db, cfg, log)

profile, err := svc.Profile(ctx, "telegram:12345")

err = svc.SetProfile(ctx, "telegram:12345", []string{
    "Prefers concise status reports",
})

results, err := svc.Search(ctx, goncho.SearchParams{
    ProfileID: "mineru",
    Peer:      "telegram:12345",
    Query:     "deployment preferences",
    Limit:     5,
})

pack, err := svc.Context(ctx, goncho.ContextParams{
    ProfileID: "mineru",
    Peer:      "telegram:12345",
    Query:     "what should I know before deploying?",
})

write, err := svc.Conclude(ctx, goncho.ConcludeParams{
    ProfileID:  "mineru",
    Peer:       "telegram:12345",
    Conclusion: "Deploy only after tests and docs build pass.",
})
```

Full API reference: [pkg.go.dev/github.com/TrebuchetDynamics/goncho](https://pkg.go.dev/github.com/TrebuchetDynamics/goncho)

---

## Gormes And Navivox Integration

If you are building on Gormes, use Goncho through the Gormes adapter instead of reaching into database internals. The adapter opens profile-local SQLite state, runs migrations, creates the service, and exposes the public memory tools.

```go
mem, err := gormesgoncho.Open(ctx, gormesgoncho.Config{
    ProfilesDirectory: ".gormes/profiles",
    ProfileID:         "mineru",
    WorkspaceID:       "gormes-prod",
    ObserverID:        "gormes",
})
```

Register these with the Gormes tool registry:

```text
goncho_context
goncho_search
goncho_remember
goncho_review
goncho_handoff
```

`goncho_context` has public E2E coverage for generated primer behavior under `max_tokens`: it preserves the newest in-budget turns and excludes older turns outside the budget while returning a representation for the target peer.

For Navivox, the same boundary applies: the Android app should treat Goncho as the local memory kernel behind the phone-hosted Gormes runtime, not as a separate memory server users must operate.

See: [Gormes Agent Integration](docs-site/src/content/docs/integrations/gormes-agent.md)

---

## Multi-Profile Memory

Gormes can manage multiple profiles in one runtime. Goncho supports that at the contract and API level.

Memory visibility is determined by:

```text
workspace_id + profile_id + scope + peer_id
```

Default behavior:

- If `profile_id` is present and `scope` is empty, Goncho uses private `profile` scope.
- Workspace-shared recall requires explicit `scope: "workspace"`.
- Profile A cannot read profile B memory by default.

Gormes profile-local state can live under a custom profile root:

```text
.gormes/profiles/<profile_id>/goncho.db
.gormes/profiles/<profile_id>/GONCHO_MEMORY.md
```

Gormes adapter example:

```go
mem, err := gormesgoncho.Open(ctx, gormesgoncho.Config{
    ProfilesDirectory: ".gormes/profiles",
    ProfileID:         "mineru",
    WorkspaceID:       "gormes-prod",
    ObserverID:        "gormes",
})
```

See: [Gormes Agent Integration](docs-site/src/content/docs/integrations/gormes-agent.md)

---

## Trust Model

Goncho separates memory into layers:

| Layer | Meaning |
| --- | --- |
| Evidence | Raw observations from sessions, tools, files, imports, and user messages. |
| Claims | Interpreted statements derived from evidence. |
| Beliefs | Scoped, time-aware memory eligible for retrieval. |
| Context packs | Compact prompt-ready orientation with warnings. |
| Review | Conflict, stale-memory, quarantine, and verification surfaces. |

Design principles:

- Evidence before belief.
- Claims, not chunks.
- Live verification before action.
- Profile isolation before recall.
- Bounded, auditable memory writes.
- Orientation packs, not memory dumps.
- Negative memory matters.
- Trust is the moat.

---

## Local Verification

Goncho favors deterministic local tests over benchmark theater.

```bash
go test ./...
```

High-signal checks:

```bash
go test ./... -run TestGormesMultiProfileMemoryIsolation
go test ./... -run TestGonchoPublicToolsRestartE2E
go test ./... -run TestGonchoGoalPromptInjectionImportIsQuarantinedE2E
go test ./... -run TestGonchoGoalStaleCodeClaimRequiresLiveVerificationE2E
go test ./... -run TestGonchoGoalNegativeDriftAnchorWarnsBeforeRepeatedFailureE2E
```

Release, benchmark, and public package smoke checks:

`make release-smoke` runs release metadata checks, ecosystem smoke, Go tests, vet, race tests, and the docs-site build before local pre-tag decisions.

```bash
make release-smoke
make ecosystem-smoke
go test ./cmd/goncho-bench
make bench-longmemeval-s-smoke
make bench-locomo-smoke
make bench-locomo
make bench-locomo-backends-smoke
make bench-locomo-backends
```

### LOCOMO Retrieval Benchmark

Goncho includes a deterministic LOCOMO retrieval harness. This evaluates retrieval only, not answer generation. It uses ID-based scoring with no LLM judge, and `answer_hint` fields are never indexed or scored.

Pinned full run evidence:

| Dataset | Questions | Memories | recall_any@5 | recall_any@10 | strict_recall@5 | strict_recall@10 | MRR |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| LOCOMO full | 1,982 | 5,882 | 60.14% | 67.91% | 51.16% | 57.67% | 46.90% |
| LOCOMO smoke | 8 | 17 | 100.00% | 100.00% | 100.00% | 100.00% | 85.42% |

Candidate-generation milestone: LOCOMO exposed a candidate-generation weakness in Goncho. After widening lexical pre-rank candidates, BM25-win `missing_candidate` failures dropped from `164` to `2`, and Goncho now essentially matches BM25 on full LOCOMO retrieval while preserving LongMemEval-S performance. This was achieved without LLM judgment, answer scoring, benchmark-specific gold-ID hacks, or ranking changes.

The full LOCOMO run compares random, recency, BM25, SQLite FTS5, and Goncho baselines against the pinned official LOCOMO source dataset.

The backend comparison harness uses the same LOCOMO JSONL, same gold IDs, same centralized scoring, same leakage checks, and same failure taxonomy for Goncho, BM25, SQLite FTS5, agentmemory, and mem0. External backends are scored only if they return stable inserted `memory_id` values. If they cannot preserve IDs, they are marked `not comparable` instead of being scored by content matching or answer text.

Current external-backend status:

| Backend | Comparable | Reason |
| --- | --- | --- |
| Goncho | yes | Native local adapter with stable IDs. |
| BM25 | yes | Local lexical baseline with stable LOCOMO IDs. |
| SQLite FTS5 | yes | Local FTS baseline with stable LOCOMO IDs. |
| agentmemory | yes, PR standalone fallback | PR #583 commit `9b18a80c9d2839b025279978d3f4b5e1f9bc6e74` preserves stable IDs via `external_id`/metadata. LOCOMO full scored `0.0000` in standalone InMemoryKV fallback mode; this is not the full running agentmemory server. |
| mem0 | no | `mem0`/`mem0ai` is not installed in the local benchmark environment; no stable-ID run was produced. |

- Milestone note: [docs/benchmarks/MILESTONE-LOCOMO-CANDIDATE-GENERATION.md](docs/benchmarks/MILESTONE-LOCOMO-CANDIDATE-GENERATION.md)
- Full report: [docs/benchmarks/locomo-2026-05-20.md](docs/benchmarks/locomo-2026-05-20.md)
- Smoke report: [docs/benchmarks/locomo-smoke.md](docs/benchmarks/locomo-smoke.md)
- Dataset notes: [docs/benchmarks/LOCOMO-DATASET.md](docs/benchmarks/LOCOMO-DATASET.md)
- External backend adapter notes: [docs/benchmarks/external-backend-adapters.md](docs/benchmarks/external-backend-adapters.md)
- Backend comparison report: [docs/benchmarks/locomo-backend-comparison.md](docs/benchmarks/locomo-backend-comparison.md)
- JSON evidence: [docs/benchmarks/results/locomo-2026-05-20-goncho.json](docs/benchmarks/results/locomo-2026-05-20-goncho.json)
- Backend comparison JSON: [docs/benchmarks/results/locomo-backend-comparison.json](docs/benchmarks/results/locomo-backend-comparison.json)

Retrieval benchmark docs: [docs-site/src/content/docs/reference/retrieval-benchmarks.md](docs-site/src/content/docs/reference/retrieval-benchmarks.md)

---

## Documentation

Start here:

- [Current Capabilities](docs-site/src/content/docs/start/current-capabilities.md)
- [Quick Start](docs-site/src/content/docs/start/quick-start.md)
- [Core API](docs-site/src/content/docs/reference/core-api.md)
- [Memory Tools](docs-site/src/content/docs/reference/memory-tools.md)
- [Gormes Agent Integration](docs-site/src/content/docs/integrations/gormes-agent.md)
- [Honcho Compatibility](docs-site/src/content/docs/reference/honcho-compatibility.md)
- [Local Markdown Memory](docs-site/src/content/docs/reference/local-markdown-memory.md)
- [Operator Runbook](docs-site/src/content/docs/operators/runbook.md)
- [Architecture Direction](docs-site/src/content/docs/roadmap/architecture-direction.md)

Research:

- [Memory Systems Metaanalysis](docs/opensource-memory-systems/METAANALYSIS-MEMORY-SYSTEMS.md)
- [Agentmemory Reference](docs/opensource-memory-systems/agentmemory/README.md)

Docs site:

```bash
cd docs-site
npm ci
npm run dev
```

Build:

```bash
cd docs-site
npm run build
```

---

## Repository Map

| Path | Purpose |
| --- | --- |
| `service.go` | Main embedded service API. |
| `types.go` | Public request/result contracts. |
| `sql.go` | SQLite-backed profile, conclusion, and session operations. |
| `observations.go` | Raw evidence capture and audit-backed observations. |
| `goncho_public_tools.go` | Public agent-facing tool surface. |
| `memory_tools.go` | Generic MCP-style memory tools. |
| `review.go` / `review_tool.go` | Review queue behavior. |
| `code_claim_verification.go` | Live verification for remembered code/file claims. |
| `drift_anchor.go` | Negative-memory drift warning logic. |
| `integration/gormes/` | Gormes adapter. |
| `http/` | Local HTTP adapter and compatibility tests. |
| `docs-site/` | Starlight documentation site. |
| `docs/opensource-memory-systems/` | Research corpus and reference systems. |

---

## Development

```bash
git clone https://github.com/TrebuchetDynamics/goncho.git
cd goncho
go test ./...
```

If tests fail, treat the first compiler or test error as the source of truth. Goncho intentionally favors evidence over optimistic status claims.

## License

MIT

---

Goncho is developed by [Trebuchet Dynamics](https://github.com/TrebuchetDynamics) as part of a local-first agent infrastructure ecosystem.

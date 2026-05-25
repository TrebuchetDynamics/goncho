# Goncho

<p align="center">
  <strong>Context architecture for long-horizon AI agents.</strong><br/>
  Persistent, temporal, evidence-backed memory without stuffing endless context into prompts.
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/TrebuchetDynamics/goncho/service"><img src="https://pkg.go.dev/badge/github.com/TrebuchetDynamics/goncho/service.svg" alt="Go Reference" /></a>
  <a href="https://github.com/TrebuchetDynamics/goncho/releases/tag/v0.3.0"><img src="https://img.shields.io/badge/release-v0.3.0-blue" alt="Release v0.3.0" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License: MIT" /></a>
</p>

<p align="center">
  <a href="#why">Why</a> &bull;
  <a href="#architecture">Architecture</a> &bull;
  <a href="#core-concepts">Concepts</a> &bull;
  <a href="#retrieval-benchmarks">Benchmarks</a> &bull;
  <a href="#quick-start">Quick Start</a> &bull;
  <a href="#core-api">API</a>
</p>

---

Goncho gives agents **trust-preserving context architecture**: active memory, retrieval provenance, temporal validity, negative memory, scoped beliefs, and local-first storage for Go-native runtimes.

> Goncho does not treat memory as a vector database problem.
> It treats memory as a claims-and-evidence system.

## Why

Most AI memory systems fail in the same way: they retrieve old text as if it were current truth.

They often:

- dump raw history into prompts;
- retrieve stale information without temporal validity;
- lose provenance between a claim and the evidence behind it;
- repeat failed approaches because dead ends are not first-class memory;
- cannot distinguish “was true then” from “is safe to act on now.”

Goncho solves this with scoped memory, evidence-backed claims, hybrid retrieval, temporal metadata, active memory consolidation, negative memory, and trust-aware context generation.

The goal is not “the agent remembers everything.”

The goal is:

```text
the agent remembers correctly
```

## Architecture

Goncho makes the memory pipeline explicit:

```text
Raw Evidence
    ↓
Claims + Temporal Metadata
    ↓
Scoped Beliefs + Review State
    ↓
Hybrid Retrieval (SQLite FTS + optional semantic vector hits + graph/provenance signals)
    ↓
Orientation Pack Generation
    ↓
Agent Runtime Context
    ↓
Reflection / Consolidation / Decay / Negative Memory
```

That architecture is why Goncho feels different from a vector wrapper. Search can find candidate memories; it does not decide truth. Context packs orient an agent; they do not authorize action. Review queues, stale-claim warnings, and provenance keep the host runtime in control.

## Core Concepts

### Claims + Evidence

Every durable memory in Goncho is designed to be:

- a claim, not just a chunk;
- linked to evidence;
- scoped to workspace, profile, peer, and session context;
- temporally bounded or revisable when truth changes;
- retrievable with provenance and scoring diagnostics.

### Negative Memory

Goncho remembers what *not* to repeat:

- dead ends;
- failed attempts;
- rejected plans;
- drift anchors;
- contradictory evidence;
- stale code claims that require live verification.

### Orientation Packs

Agents should not boot with raw dumps of everything they have ever seen. Goncho builds orientation packs with:

- current goals;
- trusted facts;
- recent episodes;
- known preferences;
- warnings;
- unresolved conflicts;
- remembered dead ends.

The output is compact context for action, not a permission slip to skip live checks.

## Agent Interface

Goncho intentionally exposes a small memory surface to agent hosts:

| Surface | Purpose |
| --- | --- |
| `context` | Build an orientation pack before an agent acts. |
| `search` | Retrieve scoped memory candidates. |
| `remember` | Store evidence-backed memories through the public tool boundary. |
| `review` | Inspect conflicts, stale claims, quarantine, and pending decisions. |
| `handoff` | Produce continuation context for another agent or session. |

Complexity stays internal: temporal metadata, query expansion, vector fusion, graph/provenance signals, review state, and consolidation are implementation details behind a small host-facing API.

## Retrieval Benchmarks

Goncho ships benchmark code and frozen evidence because memory claims should be reproducible.

**LongMemEval-S** retrieval run, 500 questions, retrieval-only evaluation — no LLM reader, no LLM judge:

| System | recall_any@5 | recall_any@10 | MRR |
| --- | ---: | ---: | ---: |
| agentmemory BM25+Vector reference | 95.20% | 98.60% | 88.20% |
| Goncho 2026-05-20 run | 96.80% | 98.00% | 91.35% |

Scope matters: this evaluates retrieval over long conversational memory, not end-to-end QA. The frozen report documents leakage checks, source provenance, and the exact command.

Reproduce the CI-safe version:

```bash
make bench-longmemeval-s-smoke
```

Run the full pinned benchmark when you have the dataset prepared:

```bash
make bench-longmemeval-s
```

Goncho also includes LOCOMO retrieval and backend-comparison harnesses with stable inserted IDs, leakage checks, failure buckets, and no answer-text rescue. See [Retrieval Benchmarks](docs-site/src/content/docs/reference/retrieval-benchmarks.md) for methodology, the external adapter contract, and current agentmemory PR #583 stable-ID status.

## Philosophy

Memory is not storage.

Memory is:

- selective;
- temporal;
- revisable;
- uncertain;
- evidence-backed;
- self-regulating.

Vectors are useful. They are not truth. The agent should remember enough to orient itself, verify enough to act safely, and forget or revise enough to avoid accumulating confident nonsense.

## Local-first By Default

Goncho is designed for:

- offline memory;
- private context;
- edge inference;
- inspectable SQLite state;
- reproducible retrieval;
- Go-native hosts on workstations, small servers, WSL2, macOS, Windows, Linux, and Termux.

Users should not have to trust a black-box cloud memory layer just to give an agent continuity.

## Positioning

Great open-source memory projects set useful expectations. **[mem0](https://github.com/mem0ai/mem0)** popularized simple, product-like agent memory APIs. **[agentmemory](docs/opensource-memory-systems/agentmemory/README.md)** demonstrates the power of broad hooks, MCP surfaces, multi-agent workflows, and benchmark-driven claims.

Goncho is the trust-preserving Go-native layer for hosts that want local memory and clear action boundaries:

| If you like... | Goncho's answer |
| --- | --- |
| mem0-style simple memory APIs | A compact embedded Go service with `NewService`, `Search`, `Recall`, `Context`, and `Conclude`. |
| agentmemory-style hooks and integration | Host-neutral hook capture plus public memory tools, without forcing a separate server. |
| Vector search | Optional local `Config.VectorStore` fusion with lexical and graph signals; vectors help retrieval but do not become truth. |
| Persistent agent memory | SQLite-local evidence, claims, scoped temporal beliefs, memory slots, review queues, and deterministic migrations. |
| Production safety | Verification-first context packs, stale-code-claim warnings, prompt-injection quarantine, profile isolation, and audit-friendly recall traces. |

## Install

Use Goncho as an embedded Go module:

```bash
go get github.com/TrebuchetDynamics/goncho/service@latest
```

The service package is a library package, not a root `go install` target. The installable command is the reproducible benchmark CLI:

```bash
go install github.com/TrebuchetDynamics/goncho/cmd/goncho-bench@latest
```

From a checkout, verify the public package, local docs, external importability, and benchmark CLI readiness:

```bash
make ecosystem-smoke
```

Public `@latest` currently resolves to v0.3.0, published May 25, 2026. For production, pin the service package version rather than treating `@latest` as a deployment lock.

## At a Glance

If you are evaluating Goncho on pkg.go.dev, start here:

- **Install:** `go get github.com/TrebuchetDynamics/goncho/service@latest`.
- **Use when:** your Go agent host needs local SQLite memory, scoped recall, review queues, stale-claim warnings, and verification-first context assembly.
- **Do not use as:** a hosted memory API, generic vector database, standalone CLI app, or replacement for live checks before tool execution.
- **First useful call:** wire `memory.OpenSqlite`, run `goncho.RunMigrations`, create `goncho.NewService`, then call `svc.Context` to build an orientation pack.
- **Runnable reference:** pkg.go.dev renders the compiled `NewService` example plus compiled `Service.Context`, `Service.Search`, and `Service.Recall` examples from this module, so setup, orientation packs, scoped retrieval, and auditable recall traces are checked by `go test` instead of drifting as prose.
- **Trust boundary:** Goncho can remember, rank, and warn; the host agent must still verify file paths, APIs, credentials, and deployment state before acting.
- **What to read next:** use [Quick Start](#quick-start) for a runnable service shape, [Core API](#core-api) for common calls, and [Package Status](#package-status) for release and smoke-test evidence.

## Trust Boundary for Agent Hosts

Goncho can orient the agent by storing evidence, ranking scoped memory, assembling context packs, and warning when remembered claims may be stale. The host remains authoritative for decisions that require current state or external authority:

- Authorization and policy decisions still belong to the host runtime, gateway, or operator.
- Live filesystem, API, deployment, and credential state must be checked at action time.
- Money movement, destructive writes, and external side effects require explicit host-side gates.
- Treat retrieved memory as evidence to check, not as permission to skip live verification.

## API Map for pkg.go.dev Readers

| If you need to... | Start with | Why |
| --- | --- | --- |
| Open local memory | `memory.OpenSqlite` plus `goncho.RunMigrations` | Creates the SQLite store and schema Goncho expects. |
| Embed the service | `goncho.NewService` | Gives your Go host the profile, search, recall, context, chat, and conclude APIs. |
| Store durable facts | `svc.SetProfile`, `svc.Conclude`, or memory slots | Separates stable profile facts, current conclusions, and named durable facts/preferences. |
| Manage named slots | `CreateMemorySlot`, `GetMemorySlot`, `ListMemorySlots`, `AppendMemorySlot`, `ReplaceMemorySlot`, `DeleteMemorySlot` | Provides scoped slot memory with revisioning, tombstones, audit observations, and profile isolation. |
| Consolidate locally | `ExecuteFourTierConsolidation` | Explicitly writes working, episodic, semantic, and procedural consolidation memories with provenance. |
| Coordinate local actions | `UpsertAction`, `ReadActionGraph`, `CompleteAction`, `SignalAction` | Tracks local dependencies, frontier, next action, and coordination signals without server leases. |
| Export snapshots | `ExportSnapshotManifest`, `DiffSnapshotManifests`, `BuildSnapshotRollbackMetadata` | Produces deterministic manifests and rollback metadata while leaving git operations adapter-owned. |
| Store image refs | `StoreImageMemory`, `SearchImageMemories` | Stores image references, checksums, alt text, and metadata with embeddings explicitly deferred for later vision search. |
| Retrieve scoped memory | `svc.Search` | Returns peer/profile/session-scoped hits before you decide what to verify; transparent synonym expansion is surfaced as hit provenance. |
| Audit recall scoring | `svc.Recall` | Returns the scored `RecallTrace` with candidates, selected/rejected memories, warnings, and provenance, including query-expansion evidence, before any projection. |
| Plug semantic retrieval | `Config.VectorStore` | Optionally lets a host provide local embedding/vector hits; Goncho fuses them as `semantic` provenance with lexical and graph signals through recall RRF. |
| Build an action primer | `svc.Context` | Produces an orientation pack; hosts still verify live state before acting. |
| Capture host hooks | `svc.CaptureHostHook` | Maps host-neutral prompt, assistant, PostToolUse, failure, compact, and session lifecycle events into `Observe`, `CreateMessages`, and session summaries. |
| Discover resources/prompts | `NewMemoryResourceRegistry` | Exposes Go-neutral status, profile, latest-memory, graph-stat resources and a recall prompt without requiring an MCP server. |
| Expose agent tools | `NewGonchoContextTool`, `NewGonchoSearchTool`, `NewGonchoRecallTool`, `NewGonchoRememberTool`, `NewReviewTool`, `NewGonchoHandoffTool` | Keeps host integrations on the public tool boundary instead of database internals. |
| Reproduce retrieval evidence | `go install github.com/TrebuchetDynamics/goncho/cmd/goncho-bench@latest` | Installs the benchmark CLI shipped with the public module. |

## Import Path Guide for pkg.go.dev Readers

| Import path | Role | Use it for |
| --- | --- | --- |
| `github.com/TrebuchetDynamics/goncho/service` | Service library package | `RunMigrations`, `NewService`, service calls, context/search/conclude params, and public tool constructors. |
| `github.com/TrebuchetDynamics/goncho/memory` | SQLite opener | `memory.OpenSqlite` when you want a local file-backed store for an embedded host. |
| `github.com/TrebuchetDynamics/goncho/cmd/goncho-bench` | Command only | `go install .../cmd/goncho-bench@latest` for reproducible retrieval reports; do not import it into an agent host. |
| `github.com/TrebuchetDynamics/goncho/memorymirror` | Architecture mirror/port matrix | Inspect the Go-native mirror of the upstream broad-memory reference at `355124141625ccc0d740ae08ddaaf77fe2c165ae`: pipeline stages, memory tiers, retrieval streams, hooks, MCP tools, Goncho seams, and explicit residual gaps. |

Stay on public service and tool APIs first. If pkg.go.dev shows a lower-level type before the service examples, treat it as implementation detail until `NewService`, `svc.Context`, `svc.Search`, `svc.Recall`, or a public tool constructor cannot express the host need.

## Minimal Embedded Skeleton

Copy this skeleton into a new Go module when you want the shortest local-memory host shape rather than the benchmark CLI:

```go
package main

import (
    "context"
    "fmt"
    "os"
    "path/filepath"

    "github.com/TrebuchetDynamics/goncho/service"
    "github.com/TrebuchetDynamics/goncho/memory"
)

func main() {
    ctx := context.Background()
    dir := "."
    if len(os.Args) > 1 {
        dir = os.Args[1]
    }

    store, err := memory.OpenSqlite(filepath.Join(dir, "goncho.db"), 0, nil)
    if err != nil {
        panic(err)
    }
    defer func() { _ = store.Close(ctx) }()

    if err := goncho.RunMigrations(store.DB()); err != nil {
        panic(err)
    }

    svc := goncho.NewService(store.DB(), goncho.Config{
        WorkspaceID:    "local-agent",
        ObserverPeerID: "assistant",
    }, nil)

    pack, err := svc.Context(ctx, goncho.ContextParams{
        Peer:      "local-user",
        Query:     "what should I verify before acting?",
        MaxTokens: 2000,
    })
    if err != nil {
        panic(err)
    }
    fmt.Println(pack.Representation) // orientation pack, not permission to skip live checks
}
```

## Host Integration Checklist

Use this checklist when embedding Goncho in an agent host after the minimal skeleton:

- Open SQLite with `memory.OpenSqlite` and close the store during host shutdown.
- Run `goncho.RunMigrations` before `goncho.NewService` on every boot; migrations are the schema contract the service expects.
- Set `WorkspaceID` and `ObserverPeerID` so memory, review queues, and audit output are attributable to the host.
- Pass explicit `ProfileID`, `Peer`, and `SessionKey` on context, search, and conclude calls when the host has profile or session routing.
- Call `svc.Context` before tool execution to build an orientation pack, then let the host decide which live checks are required.
- Write conclusions with evidence after observations, user-visible decisions, or verified tool results; avoid storing guesses as durable claims.
- Verify live state before acting: file paths, APIs, credentials, deployment state, and external services still need current proof outside memory.

## Package Status

Goncho is pre-1.0, but it has the public release signals needed to evaluate it as an ecosystem component: a tagged v0.3.0 release published May 25, 2026, a valid Go module, pkg.go.dev API docs, public docs, reproducible benchmark commands, deterministic benchmark methodology, and stable-ID backend comparison artifacts. The LOCOMO backend comparison is conversation-scoped so duplicate content in other conversations cannot win by content alone. LOCOMO backend-comparison reports expose stable-ID failure buckets through the JSON `failure_buckets` field and markdown `Failure buckets` table, beside rank-based `failure_categories`, without changing scoring or regenerating frozen LOCOMO artifacts. Benchmark methodology, the external adapter contract, and current agentmemory PR #583 stable-ID status live in [Retrieval Benchmarks](docs-site/src/content/docs/reference/retrieval-benchmarks.md).

### go.dev Signal Map

| go.dev signal | Current state | Local proof |
| --- | --- | --- |
| Version | `v0.3.0 / Latest`, published May 25, 2026 | `make public-release-smoke` checks public `@latest` metadata with `go list -m -json`. |
| Valid go.mod file | Module path is `github.com/TrebuchetDynamics/goncho` | `make local-module-smoke` checks `go list -m -json` for module path and Go version. |
| Redistributable license | `MIT` | License file is checked in and pkg.go.dev marks it redistributable. |
| Package documentation | Root package docs render with examples | `make package-doc-smoke` checks `go doc .`; compiled examples run in `go test ./...`. |
| External importability | Public module can be imported from a temporary module | `make public-module-smoke` runs `go get github.com/TrebuchetDynamics/goncho/service@latest` and compiles a minimal service import. |
| Command install path | Root is a library; `cmd/goncho-bench` is the command | `make install-smoke` installs checkout-local `./cmd/goncho-bench` and starts `--help`. |
| Imported by | Imported by count is an adoption signal, not a local correctness gate | Prefer the smoke commands above for reproducible package-readiness evidence. |

### Versioning and Adoption Notes

- **Pin production dependencies:** Goncho is pre-1.0 stability software. For reproducible builds, use `go get github.com/TrebuchetDynamics/goncho/service@v0.3.0` or a reviewed commit; do not treat `@latest` as a deployment lock.
- **Read adoption counters carefully:** pkg.go.dev currently shows Imported by 0. That reverse-dependency count is not a correctness gate; use it as adoption context, then rely on the smoke checks below for package readiness.
- **Upgrade by evidence:** before changing the pinned version, run `make ecosystem-smoke` from a checkout and keep host-side live verification in place.

Verify public release metadata, local Go module metadata, package documentation, public docs site build, external importability, and the checkout-local benchmark CLI without editing another project:

```bash
make ecosystem-smoke
```

For one CI-safe checkout gate that proves the benchmark CLI starts, core package behavior passes, static checks pass, and tiny retrieval/BEAM benchmark paths run end to end, use:

```bash
make stable-e2e-bench-smoke
```

That target runs `install-smoke`, `go test ./...`, `go vet ./...`, and benchmark smoke paths equivalent to `bench-longmemeval-s-smoke`, `bench-locomo-smoke`, and `bench-beam-smoke` with temporary outputs so the checkout stays clean.

For the narrower public release metadata proof only, run `make public-release-smoke`; it checks the documented public `@latest` version and published date from `go list -m -json`. For the narrower local go.mod metadata proof only, run `make local-module-smoke`; it checks the module path and Go version from `go list -m -json`. For the narrower package documentation proof only, run `make package-doc-smoke`; it checks that local package docs render through `go doc .`. For the narrower public docs site proof only, run `make docs-site-smoke`; it checks the local docs-site build with `npm run build`. For the narrower external import proof only, run `make public-module-smoke`. For the CI-safe external backend comparison proof, run `make bench-locomo-backends-smoke`.

Use `go get github.com/TrebuchetDynamics/goncho/service@latest` to depend on the library package. For the command-line benchmark runner, use `go install github.com/TrebuchetDynamics/goncho/cmd/goncho-bench@latest` or checkout-local `make install-smoke`.

---

## Why Goncho

Most agent memory systems optimize for breadth: more connectors, more tools, more autonomous behavior. Goncho optimizes for trust: memory correctness, bounded writes, reproducible retrieval, local state, and verification before action.

Goncho exists because Gormes and Navivox need memory that can run anywhere the agent runs: a workstation, small server, Windows laptop, WSL2 shell, macOS terminal, or Android phone through Termux. The memory layer cannot assume Python packaging, Docker, Redis, hosted vector infrastructure, or always-online cloud services.

Goncho is inspired by broad integration systems like [`agentmemory`](docs/opensource-memory-systems/agentmemory/README.md), and the public `memorymirror` package now carries a source-pinned architecture mirror/port matrix for upstream `https://github.com/rohitg00/agentmemory` commit `355124141625ccc0d740ae08ddaaf77fe2c165ae` without adopting the upstream project name as Goncho API surface. Use `memorymirror.ArchitectureManifest()` to inspect which upstream pipeline stages, four memory tiers, retrieval streams, hooks, and 53 MCP tools are delivered through Goncho seams versus partial, adapter-owned, deferred, or explicitly excluded. Use `memorymirror.NewToolRegistry` when a host wants compatible executable aliases (`memory_save`, `memory_smart_search`, `memory_recall`, `memory_profile`) backed by Goncho's local service APIs.

Goncho still makes a different product bet:

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

    "github.com/TrebuchetDynamics/goncho/service"
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

trace, err := svc.Recall(ctx, goncho.RecallQuery{
    Peer:  "telegram:12345",
    Query: "deployment preferences",
    Limit: 5,
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

Full API reference: [pkg.go.dev/github.com/TrebuchetDynamics/goncho/service](https://pkg.go.dev/github.com/TrebuchetDynamics/goncho/service)

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
goncho_recall
goncho_remember
goncho_review
goncho_handoff
```

`goncho_context` has public E2E coverage for generated primer behavior under `max_tokens`: it preserves the newest in-budget turns and excludes older turns outside the budget while returning a representation for the target peer. `goncho_recall` exposes the scored recall trace, compact diagnostics report, formatted diagnostics text, and deterministic replay contract through the same public tool seam; pass `compact: true` to keep diagnostics while omitting full trace/replay payloads. The Gormes adapter `Status()` includes a compact capability summary plus registered tool operation specs with JSON-friendly lowercase fields so hosts can log/discover schemas without reaching into tool instances; use `Status().RequireCapabilities(...)` for startup health gates.

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

LOCOMO benchmark scope: retrieval-only; no answer generation, no LLM judge, ID-based scoring, and `answer_hint` fields are never indexed or scored.

Pinned full run evidence:

| Dataset | Questions | Memories | recall_any@5 | recall_any@10 | strict_recall@5 | strict_recall@10 | MRR |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| LOCOMO full | 1,982 | 5,882 | 60.14% | 67.91% | 51.16% | 57.67% | 46.90% |
| LOCOMO smoke | 8 | 17 | 100.00% | 100.00% | 100.00% | 100.00% | 85.42% |

Result artifacts should not stop at recall and MRR. Current smoke and backend-comparison artifacts also report `NDCG@5`, `NDCG@10`, latency min/p50/p95/max, RSS, database size, memory token estimate, Top-K, failure categories, and leakage checks. Preserve the frozen historical full-run evidence at `docs/benchmarks/results/locomo-2026-05-20-goncho.json`; it is not regenerated by smoke targets. Treat regenerated smoke and backend-comparison artifacts as fresh harness checks, and remember latency/RSS measurements are host- and run-sensitive; compare backend evidence through `docs/benchmarks/results/locomo-backend-comparison.json`.

- Full LOCOMO reproduction: `make bench-locomo` — manual full run with pinned conversion; writes date-stamped full-run artifacts.
- Retrieval smoke reproduction: `make bench-locomo-smoke` — CI-safe tiny fixture for retrieval report regeneration.
- Backend smoke reproduction: `make bench-locomo-backends-smoke` — CI-safe external-backend harness check.
- Full backend comparison reproduction: `make bench-locomo-backends` — manual full backend comparison with local adapter prerequisites.

Candidate-generation milestone: LOCOMO exposed a candidate-generation weakness in Goncho. After widening lexical pre-rank candidates, BM25-win `missing_candidate` failures dropped from `164` to `2`, and Goncho now essentially matches BM25 on full LOCOMO retrieval while preserving LongMemEval-S performance. This was achieved without LLM judgment, answer scoring, benchmark-specific gold-ID hacks, or ranking changes.

Full LOCOMO baseline set: random, recency, BM25, SQLite FTS5, and Goncho in the frozen full LOCOMO run against the pinned official LOCOMO source dataset.

LOCOMO source provenance: `https://github.com/snap-research/locomo` at revision `3eb6f2c585f5e1699204e3c3bdf7adc5c28cb376`. Source SHA256: `79fa87e90f04081343b8c8debecb80a9a6842b76a7aa537dc9fdf651ea698ff4`. License note: Creative Commons Attribution-NonCommercial 4.0 International (CC BY-NC 4.0).

LOCOMO converted dataset evidence: memories at `data/locomo/memories.jsonl`, questions at `data/locomo/questions.jsonl`. Questions: `1982`. Memories: `5882`.

LOCOMO leakage check counts: Answer text present in memory content: `3026`; Gold IDs present in memory content: `0`; Question text present in memory content: `0`. Answer-text presence is reported because LOCOMO answers may be literal spans from the gold memories, while `answer_hint` fields are never indexed or scored.

LOCOMO category metric groups: `adversarial_unanswerable`, `multi_hop_retrieval`, `open_domain_retrieval`, `single_hop_retrieval`, and `temporal_retrieval`.

LOCOMO category question counts:

- `adversarial_unanswerable`: `446` questions
- `multi_hop_retrieval`: `92` questions
- `open_domain_retrieval`: `841` questions
- `single_hop_retrieval`: `282` questions
- `temporal_retrieval`: `321` questions

LOCOMO Goncho category metrics:

- `adversarial_unanswerable`: recall_any@5 `61.66%`, recall_any@10 `71.52%`, MRR `48.90%`
- `multi_hop_retrieval`: recall_any@5 `35.87%`, recall_any@10 `41.30%`, MRR `24.76%`
- `open_domain_retrieval`: recall_any@5 `63.73%`, recall_any@10 `70.27%`, MRR `50.39%`
- `single_hop_retrieval`: recall_any@5 `47.16%`, recall_any@10 `59.22%`, MRR `31.91%`
- `temporal_retrieval`: recall_any@5 `66.98%`, recall_any@10 `71.96%`, MRR `54.47%`

LOCOMO Goncho strict category metrics:

- `adversarial_unanswerable`: strict_recall@5 `60.09%`, strict_recall@10 `69.73%`
- `multi_hop_retrieval`: strict_recall@5 `15.22%`, strict_recall@10 `18.48%`
- `open_domain_retrieval`: strict_recall@5 `60.76%`, strict_recall@10 `67.54%`
- `single_hop_retrieval`: strict_recall@5 `9.22%`, strict_recall@10 `13.48%`
- `temporal_retrieval`: strict_recall@5 `60.75%`, strict_recall@10 `65.11%`

LOCOMO BM25 category metrics:

- `adversarial_unanswerable`: recall_any@5 `61.88%`, recall_any@10 `71.52%`, MRR `48.92%`
- `multi_hop_retrieval`: recall_any@5 `35.87%`, recall_any@10 `41.30%`, MRR `24.76%`
- `open_domain_retrieval`: recall_any@5 `63.97%`, recall_any@10 `70.27%`, MRR `50.35%`
- `single_hop_retrieval`: recall_any@5 `46.81%`, recall_any@10 `58.87%`, MRR `31.71%`
- `temporal_retrieval`: recall_any@5 `66.67%`, recall_any@10 `72.59%`, MRR `54.60%`

LOCOMO SQLite FTS5 category metrics:

- `adversarial_unanswerable`: recall_any@5 `51.12%`, recall_any@10 `58.97%`, MRR `39.09%`
- `multi_hop_retrieval`: recall_any@5 `30.43%`, recall_any@10 `36.96%`, MRR `20.42%`
- `open_domain_retrieval`: recall_any@5 `52.68%`, recall_any@10 `60.05%`, MRR `41.87%`
- `single_hop_retrieval`: recall_any@5 `35.11%`, recall_any@10 `45.39%`, MRR `25.38%`
- `temporal_retrieval`: recall_any@5 `54.83%`, recall_any@10 `60.75%`, MRR `43.48%`

LOCOMO random baseline category metrics:

- `adversarial_unanswerable`: recall_any@5 `1.35%`, recall_any@10 `2.47%`, MRR `0.88%`
- `multi_hop_retrieval`: recall_any@5 `3.26%`, recall_any@10 `5.43%`, MRR `1.58%`
- `open_domain_retrieval`: recall_any@5 `1.19%`, recall_any@10 `2.50%`, MRR `0.89%`
- `single_hop_retrieval`: recall_any@5 `2.48%`, recall_any@10 `3.55%`, MRR `0.97%`
- `temporal_retrieval`: recall_any@5 `0.00%`, recall_any@10 `0.62%`, MRR `0.08%`

LOCOMO recency baseline category metrics:

- `adversarial_unanswerable`: recall_any@5 `0.45%`, recall_any@10 `0.45%`, MRR `0.28%`
- `multi_hop_retrieval`: recall_any@5 `0.00%`, recall_any@10 `1.09%`, MRR `0.16%`
- `open_domain_retrieval`: recall_any@5 `0.36%`, recall_any@10 `0.59%`, MRR `0.21%`
- `single_hop_retrieval`: recall_any@5 `0.71%`, recall_any@10 `1.77%`, MRR `0.29%`
- `temporal_retrieval`: recall_any@5 `0.31%`, recall_any@10 `0.93%`, MRR `0.14%`

LOCOMO improvement recommendations:

- Focus first on the weakest frozen metrics: multi-hop recall_any@10 is `41.30%`, multi-hop strict_recall@10 is `18.48%`, and single-hop strict_recall@10 is `13.48%`.
- Use hybrid candidate generation to combine local lexical hits, backend-comparison lessons, and graph-expanded evidence before top-K truncation.
- Add multi-hop graph expansion and query decomposition so relationship questions retrieve each required companion memory before final ranking.
- Improve temporal and speaker routing so changed facts and who-said-what stay in the right conversation branch.
- Add coverage-aware ranking so top-K results cover distinct gold evidence instead of near-duplicate memories.
- Keep failure-driven evaluation by tying each change to a failure-audit bucket and stable inserted `memory_id` evidence.
- Treat these as retrieval improvements, not extra tools: do not introduce answer hints, LLM judges, answer-text scoring, or benchmark-specific gold-ID hacks.

The backend comparison harness uses the same LOCOMO JSONL, same gold IDs, same centralized scoring, same leakage checks, and same failure taxonomy for Goncho, BM25, SQLite FTS5, agentmemory, and mem0. External backends are scored only if they return stable inserted `memory_id` values. If they cannot preserve IDs, they are marked `not comparable` instead of being scored by content matching or answer text.

An out-of-conversation `memory_id` is rejected before scoring and labeled `failure_bucket "wrong_branch_retrieval"`; it is not rescued by content matching or answer text.

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
- Failure audit evidence: [docs/benchmarks/failures/locomo-2026-05-20-categories.jsonl](docs/benchmarks/failures/locomo-2026-05-20-categories.jsonl) and [docs/benchmarks/failures/locomo-backend-comparison.jsonl](docs/benchmarks/failures/locomo-backend-comparison.jsonl) for retrieval-miss categories and not-comparable backend evidence.
- Candidate-generation failure comparison audit: [docs/benchmarks/failures/locomo-2026-05-20-bm25-vs-goncho.jsonl](docs/benchmarks/failures/locomo-2026-05-20-bm25-vs-goncho.jsonl) records the BM25-win `missing_candidate` diagnosis used by the milestone.
- Smoke-only failure audit evidence: [docs/benchmarks/failures/locomo-smoke-categories.jsonl](docs/benchmarks/failures/locomo-smoke-categories.jsonl) and [docs/benchmarks/failures/locomo-backend-comparison-smoke.jsonl](docs/benchmarks/failures/locomo-backend-comparison-smoke.jsonl) are harness smoke outputs, not historical full-run evidence.

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

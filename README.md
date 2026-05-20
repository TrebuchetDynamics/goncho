# Goncho

**High-trust memory runtime for Go agents.**

Goncho gives AI agents durable, auditable memory without requiring a hosted memory API, a vector database, or a pile of prompt-stuffed notes. It runs in your Go process, stores state in SQLite, and exposes Honcho-compatible primitives for profile, search, context, chat, conclusions, review, handoff, and local memory tools.

> Goncho helps agents know what they know, why they know it, when it may be stale, and what they must verify before acting.

```bash
go get github.com/TrebuchetDynamics/goncho
```

[![Go Reference](https://pkg.go.dev/badge/github.com/TrebuchetDynamics/goncho.svg)](https://pkg.go.dev/github.com/TrebuchetDynamics/goncho)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

---

## At a Glance

| Question | Answer |
| --- | --- |
| What is it? | A high-trust memory runtime for Go agent hosts. |
| Storage | SQLite by default. No required hosted service. |
| Agent surface | Context, search, remember, review, handoff, and Honcho-compatible primitives. |
| Core idea | Evidence before belief; live verification before action. |
| Multi-profile contract | `workspace_id + profile_id + scope + peer_id` determines memory visibility; `profile_directory` records where profile-local state lives. |
| Trust model | Surface provenance, staleness, quarantine, review, and verification warnings. |
| Best fit | Coding agents, private assistants, MCP hosts, and long-running local agents. |
| Verification | Deterministic local E2E tests with temporary SQLite, local files, and `httptest`. |

## Contents

- [Why Goncho](#why-goncho)
- [What Goncho Provides Today](#what-goncho-provides-today)
- [When to Use Goncho](#when-to-use-goncho)
- [Quick Start](#quick-start)
- [Core API Shape](#core-api-shape)
- [Trust-Preserving Memory Model](#trust-preserving-memory-model)
- [Real Local E2E Proof](#real-local-e2e-proof)
- [Proof Matrix](#proof-matrix)
- [Architecture Data Flow](#architecture-data-flow)
- [Honcho and MCP Compatibility](#honcho-and-mcp-compatibility)
- [Research Grounding](#research-grounding)
- [Current Status](#current-status)
- [Roadmap](#roadmap)
- [Repository Guide](#repository-guide)

---

## Why Goncho

The crowded path for agent memory is broader integration, more tools, and more autonomy theater. Goncho takes a narrower infrastructure path: memory correctness, operational trust, verified state, bounded behavior, and retrieval discipline.

Goncho is inspired by broad integration systems like [`agentmemory`](docs/opensource-memory-systems/agentmemory/README.md), but it makes a different product bet:

```text
agentmemory: broad integration layer
Goncho:      high-trust memory runtime
```

That means fewer tools by default, stronger trust semantics, bounded memory writes, reproducible retrieval, local inspectable state, and one hard rule for engineering agents:

```text
live verification before action
```

If memory says a file exists, verify it. If memory says Redis is installed, verify it. If memory says the user approved a migration, verify it. If memory says an API path still exists, verify it. Goncho treats memory as orientation until evidence proves it is safe to act.

Most agent memory systems start as retrieval systems:

```text
old text -> vector search -> top-k chunks -> prompt
```

That is not enough for long-running engineering agents. Real memory has failure modes:

- A fact was true last week but is stale now.
- A file moved, but old memory still mentions the old path.
- A previous fix failed repeatedly and should not be retried blindly.
- A memory came from an untrusted import and should not steer behavior.
- A useful conclusion needs scope: user, repo, session, project, or workspace.
- Private developer workflows cannot depend on a remote memory service.

Goncho treats memory as **trust-preserving context architecture**:

```text
raw evidence
  -> claims
  -> scoped temporal beliefs
  -> task-specific orientation
  -> agent action
  -> review, verification, revision, or forgetting
```

Vectors are useful. Search is useful. Goncho does not make either one the source of truth. The source of truth is local, auditable memory with scope, provenance, lifecycle state, and verification warnings. Retrieval can suggest; verification decides.

## What Goncho Provides Today

| Capability | What it means |
| --- | --- |
| **Embedded Go service** | Use Goncho as a library inside your agent host. |
| **SQLite by default** | Durable local memory with no required network service, sidecar, or cloud dependency. |
| **Honcho-compatible primitives** | Profile, search, context, chat, conclude, reasoning-style compatibility surfaces. |
| **MCP-style memory tools** | `store_memory`, `retrieve_memory`, `update_memory`, `summarize_memories`, and `forget_memory`. |
| **Public Goncho tools** | Stable agent-facing tools for context, search, remember, review, and handoff. |
| **Context packs** | Build compact prompt-ready orientation instead of dumping every matching memory. |
| **Multi-profile memory isolation** | Gormes-style multi-profile hosts can pass `profile_id` so profile-private memories do not bleed across agents. |
| **Evidence and claims** | Preserve raw observations separately from interpreted conclusions. |
| **Review queues** | Surface conflict and stale-memory items instead of silently trusting them. |
| **Prompt-injection quarantine** | Prompt-injection-like imported content is preserved as skipped evidence and excluded from trusted context. |
| **Live code-claim verification** | Check remembered file/path claims against live repo state before trusting them. |
| **Negative drift anchors** | Warn when a new prompt resembles a known failed path or dead end. |
| **Local E2E coverage** | Core behavior is tested with deterministic SQLite and `httptest` flows. |

## When to Use Goncho

Use Goncho when you are building:

- coding agents that need repo-aware memory,
- local-first assistants for private workflows,
- agent hosts that need Honcho-compatible semantics,
- MCP hosts that need durable memory tools,
- long-running agents that need reviewable beliefs, not just chat history,
- systems where stale facts, prompt injection, or repeated failed fixes are unacceptable.

Do not use Goncho if you only need a hosted vector search API or a large agent-integration catalog. Goncho is a memory runtime for agents that care about trust, locality, provenance, lifecycle, profile isolation, and verified action.

## Non-Goals

Goncho intentionally avoids several common memory-system traps:

- It is not a remote memory SaaS.
- It is not a vector database wrapper.
- It is not a dashboard-first knowledge base.
- It is not trying to expose dozens of tools to the agent.
- It is not optimizing for capability breadth over memory correctness.
- It does not treat imported text as trusted memory by default.
- It does not collapse historical truth into current truth.
- It does not ask agents to act on stale assumptions without verification.

The base workflow should stay local, auditable, and testable. Optional server, team, graph, and dashboard layers can grow around that kernel without becoming mandatory for a single developer agent.

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

    // Store durable peer knowledge.
    if err := svc.SetProfile(ctx, "telegram:12345", []string{
        "Works in finance",
        "Prefers SQLite over Postgres",
        "Wants concise answers unless deeper reasoning is requested",
    }); err != nil {
        panic(err)
    }

    // Later, in a new process using the same SQLite file, recall it.
    card, err := svc.Profile(ctx, "telegram:12345")
    if err != nil {
        panic(err)
    }

    fmt.Println(card.Card)
}
```

## Core API Shape

Four calls cover the common embedded path:

```go
svc := goncho.NewService(db, cfg, log)

// Who is this peer?
profile, err := svc.Profile(ctx, "telegram:12345")

// Remember durable profile facts.
err = svc.SetProfile(ctx, "telegram:12345", []string{
    "Go developer",
    "Prefers local-first tools",
})

// Search what Goncho knows for one Gormes profile.
results, err := svc.Search(ctx, goncho.SearchParams{
    ProfileID: "mineru",
    Query:     "database preferences",
    Limit:     5,
})

// Build prompt-ready context.
pack, err := svc.Context(ctx, goncho.ContextParams{
    ProfileID: "mineru",
    Peer:      "telegram:12345",
    MaxTokens: 8000,
})
```

Full API reference: [pkg.go.dev/github.com/TrebuchetDynamics/goncho](https://pkg.go.dev/github.com/TrebuchetDynamics/goncho)

## Trust-Preserving Memory Model

Goncho separates memory into layers that answer different trust questions.

```text
┌──────────────────────────────────────────────────────────────┐
│                         Agent Loop                           │
└───────────────┬──────────────────────────────────────────────┘
                │
                ▼
┌──────────────────────────────────────────────────────────────┐
│                         Goncho                               │
│                                                              │
│  Evidence  ->  Claims  ->  Scoped Beliefs  ->  Context Pack  │
│     │             │              │                 │          │
│     ▼             ▼              ▼                 ▼          │
│  events      conclusions      profiles        prompt-ready    │
│  tools       summaries        relations       orientation     │
│  sessions    decisions        validity        with warnings   │
│                                                              │
└───────────────────────────────┬──────────────────────────────┘
                                │
                                ▼
                         SQLite database
```

### Evidence before belief

Raw observations from sessions, tools, imports, files, and user messages are preserved before interpretation. Goncho distinguishes what was observed from what the agent should currently believe.

### Claims, not chunks

Useful memory is not just an old text span. It is a scoped claim with provenance, time, lifecycle, confidence, and evidence.

### Live verification before action

Memory is not permission to act. Before using remembered state for an irreversible or repo-sensitive action, Goncho should help the host verify live reality: current files, current APIs, current approvals, current process state, and current policy.

### Profile isolation before recall

Gormes can manage multiple profiles in one runtime. Goncho treats profile identity as part of the memory contract: `workspace_id + profile_id + scope + peer_id` determines what can be read or written. The default for requests with `profile_id` is private `profile` scope; shared workspace recall requires explicit `scope: "workspace"`.

Profile-local state can also be rooted in a custom directory. For Gormes, the expected layout is:

```text
.gormes/profiles/<profile_id>/goncho.db
.gormes/profiles/<profile_id>/GONCHO_MEMORY.md
```

This keeps contract identity and filesystem state aligned: the same `profile_id` that scopes memory also selects the profile directory.

### Orientation, not dumping

Agents need compact working context:

- current peer or project profile,
- relevant canonical facts,
- recent high-signal episodes,
- known dead ends,
- unresolved conflicts,
- stale or quarantine warnings,
- source citations and verification requirements.

## Real Local E2E Proof

Goncho favors deterministic local verification. The core suite uses temporary SQLite databases, local files, and `httptest`; it does not require a hosted Honcho service, network access, cloud embeddings, or an LLM.

```bash
go test ./...
```

High-signal E2E checks:

```bash
go test ./... -run TestLocalE2E_ServiceLifecycleBuildsContextFromPublicAPIs
go test ./... -run TestHTTPServiceRestartE2E
go test ./... -run TestGonchoPublicToolsRestartE2E
go test ./... -run TestGonchoGoalPromptInjectionImportIsQuarantinedE2E
go test ./... -run TestGonchoGoalStaleCodeClaimRequiresLiveVerificationE2E
go test ./... -run TestGonchoGoalNegativeDriftAnchorWarnsBeforeRepeatedFailureE2E
```

These tests prove that Goncho can:

- open a local SQLite store,
- run migrations,
- persist profile/session/conclusion data,
- build context and search results from exported APIs,
- restart and retain public-tool memory,
- quarantine prompt-injection-like imports,
- verify stale code claims against live repo state,
- warn before repeating known failed paths.

## Retrieval Accuracy Benchmarks

Goncho is being evaluated as a long-term memory retrieval system for agents, not just a vector store. The defensible claim is **measurably strong long-memory retrieval**, not “solves memory.”

On a pinned LongMemEval-S retrieval-only run, Goncho scored **96.40% recall_any@5**, **98.00% recall_any@10**, and **81.12% MRR** using deterministic ID-based scoring against gold evidence IDs.

| System | recall_any@5 | recall_any@10 | MRR |
| --- | ---: | ---: | ---: |
| agentmemory BM25+Vector reference | 95.20% | 98.60% | 88.20% |
| agentmemory BM25-only reference | 86.20% | 94.60% | 71.50% |
| Goncho pinned run | 96.40% | 98.00% | 81.12% |

The benchmark includes:

- pinned Hugging Face dataset revision,
- SHA256 verification,
- deterministic evidence-ID scoring,
- no LLM judge,
- random, BM25, SQLite FTS5, Goncho no-rank, and Goncho baselines,
- leakage checks,
- failure-audit JSONL,
- generated reports from machine-readable JSON.

```bash
go test ./cmd/goncho-bench
make bench-longmemeval-s-smoke
```

Manual full run from a clean checkout:

```bash
make bench-longmemeval-s
```

Confidence note: the retrieval runner is deterministic, so repeated runs should reproduce the same scores for the same code, dataset revision, and conversion. Runtime and RSS vary by machine. One official dataset case contains the exact later query text inside the gold conversation; Goncho reports this leakage instead of hiding it.

Docs:

- [Retrieval Benchmarks](docs-site/src/content/docs/reference/retrieval-benchmarks.md)
- [LongMemEval-S generated report](docs/benchmarks/longmemeval-s-2026-05-20.md)
- [Benchmark Roadmap](docs/benchmarks/ROADMAP.md)

## Proof Matrix

The metaanalysis recommends evaluating memory quality with deterministic local tests before making benchmark claims. Goncho's README links claims to checks you can run locally.

| Metaanalysis risk | Goncho behavior | Local proof |
| --- | --- | --- |
| Stale code facts | Verify remembered file/path claims against live repo state before trusting them. | `TestGonchoGoalStaleCodeClaimRequiresLiveVerificationE2E` |
| Prompt-injection persistence | Preserve suspicious imports as skipped evidence and exclude them from trusted context/search. | `TestGonchoGoalPromptInjectionImportIsQuarantinedE2E` |
| Repeated failed behavior | Match prompts against negative/dead-end memory and warn before repeating a failed path. | `TestGonchoGoalNegativeDriftAnchorWarnsBeforeRepeatedFailureE2E` |
| Restart loss | Persist public tool memory across SQLite-backed service restarts. | `TestGonchoPublicToolsRestartE2E` |
| HTTP/local service drift | Exercise Honcho-compatible HTTP lifecycle with local service handlers. | `TestHTTPServiceRestartE2E` |
| API regression | Build profile, context, search, chat, and conclusion flows through exported APIs. | `TestLocalE2E_ServiceLifecycleBuildsContextFromPublicAPIs` |

Planned features are listed as planned. Implemented claims should have a local test, a source file, or both.

## Architecture Data Flow

```text
agent host / MCP host / HTTP route
  -> Goncho service API
  -> evidence capture and metadata normalization
  -> SQLite persistence and FTS-backed lookup
  -> conclusions, profiles, reviews, and memory tools
  -> trust filters: scope, freshness, quarantine, stale-claim checks, negative anchors
  -> compact context pack for the next agent action
```

This data flow is deliberately boring at the storage layer and strict at the trust layer. The goal is not to remember everything; the goal is to return the smallest useful context that can explain why it is safe to use.

## Honcho and MCP Compatibility

Goncho is designed for agents and hosts that already speak Honcho-style memory.

External tool names preserve Honcho compatibility:

```text
honcho_profile
honcho_search
honcho_context
honcho_chat
honcho_reasoning
honcho_conclude
```

MCP-style memory contracts are available for hosts that use generic memory tools:

```text
store_memory
retrieve_memory
update_memory
summarize_memories
forget_memory
```

This lets a host start with simple memory tools and grow into richer context packs without changing the agent-facing vocabulary.

Migration guide: [docs-site/src/content/docs/reference/honcho-compatibility.md](docs-site/src/content/docs/reference/honcho-compatibility.md)

## Research Grounding

Goncho is built from the project research in [`docs/opensource-memory-systems/METAANALYSIS-MEMORY-SYSTEMS.md`](docs/opensource-memory-systems/METAANALYSIS-MEMORY-SYSTEMS.md). The metaanalysis compares open-source agent memory systems and extracts fourteen design constraints for trustworthy memory, including:

1. Local-first by default, server/team mode optional.
2. MCP-first and hook-native.
3. Small agent-facing surface, rich internal pipeline.
4. Memory records are evidence, claims, routines, alerts, or relationships.
5. Memory records are scoped and time-aware.
6. Retrieval is hybrid, budgeted, and warning-aware.
7. Context injection produces cited packs, not raw dumps.
8. Stale, conflicting, and low-confidence memories are visible.
9. Negative memory and dead ends are first-class.
10. Secrets and prompt-injection-like content are quarantined before promotion.
11. Live truth is pulled from governed tools when memory is stale.
12. Surfaced memory should explain: why this, why now, why trust it?

Goncho's implementation roadmap follows those constraints rather than treating memory as plain retrieval.

## Design Principles

| Principle | Meaning |
| --- | --- |
| **Evidence before belief** | Preserve raw events and tool outputs first; derive beliefs second. |
| **Live verification before action** | Treat remembered state as orientation until current files, tools, approvals, or policies verify it. |
| **Profile isolation before recall** | Require explicit `profile_id` and scope for multi-profile hosts so one profile cannot accidentally read another profile's memory. |
| **Profile-local directories** | Support custom profile roots such as `.gormes/profiles/<profile_id>/` for profile-owned SQLite and markdown memory files. |
| **Claims, not chunks** | Store what is believed with proof, confidence, scope, and time. |
| **Bounded memory writes** | Keep writes explicit, scoped, auditable, and reversible instead of letting agents freely rewrite their own reality. |
| **Reproducible retrieval** | Prefer deterministic local tests, cited context packs, and explainable scoring over opaque recall. |
| **Hooks over manual saves** | Capture at cognitive transition points such as session start, tool use, compaction, and stop. |
| **Orientation, not dumping** | Inject compact task context, not every semantically similar memory. |
| **Negative memory matters** | Failed paths and rejected approaches are part of intelligence. |
| **Small agent surface** | Expose stable primitives for context, search, remember, review, and handoff. |
| **Trust is the moat** | Every surfaced memory should answer: why this, why now, and why trust it? |

## Comparison

| Approach | Strength | Failure mode Goncho avoids |
| --- | --- | --- |
| Flat markdown memory | Simple and editable | Becomes token bloat without search, scope, lifecycle, or review. |
| Vector-only memory | Good fuzzy recall | Returns plausible but stale or wrong chunks. |
| Cloud memory APIs | Easy hosted setup | Adds network dependency, privacy risk, and vendor lock-in. |
| Postgres + pgvector stacks | Powerful production search | Raises the setup floor for a single local Go agent. |
| Broad integration memory, such as agentmemory | Strong hook coverage, MCP/REST reach, many agent connectors | Goncho borrows the integration lessons while keeping a smaller, stricter trust surface. |
| Goncho | Local evidence, scoped beliefs, compact context | Keeps the base workflow embedded, auditable, offline-capable, and warning-aware. |

## Current Status

Goncho is pre-1.0 software. The v0.1.x line provides the initial importable local memory kernel and compatibility layer while the architecture continues to evolve.

| Area | Status |
| --- | --- |
| Embedded Go service | Implemented and tested |
| SQLite storage and migrations | Implemented and tested |
| Peer profiles and conclusions | Implemented and tested |
| Search and context APIs | Implemented and tested |
| Multi-profile memory isolation | Implemented and tested |
| Honcho-compatible tool names | Implemented |
| MCP-style memory tool contracts | Implemented and tested |
| Public context/search/remember/review/handoff tools | Implemented and tested |
| Local markdown/import workflows | Experimental |
| Prompt-injection quarantine | Implemented and tested |
| Stale code-claim verification | Implemented and tested |
| Negative drift anchors | Implemented and tested |
| Lifecycle stewardship and review queues | Experimental |
| Graph/cognitive-map/dashboard layers | Planned |
| PostgreSQL team adapter | Planned |

## Roadmap

### Phase 1: Local Memory Kernel

- SQLite storage.
- FTS-backed search.
- Peer profiles, conclusions, summaries, and context packs.
- Honcho-compatible service and tool names.
- MCP-style memory contracts.
- Local-first operation with no mandatory external model.

### Phase 2: Lifecycle and Trust

- Temporal fields and valid intervals.
- Supersession chains.
- Claim verification for files, functions, and APIs.
- Confidence, freshness, and authority scoring.
- Review inbox for conflicts, stale facts, and duplicate memories.

### Phase 3: Graph and Cognitive Map

- Conservative entity extraction.
- Memory links and relationship-aware recall.
- Project cognitive map.
- Activation-based branch selection for context packs.

### Phase 4: Drift and Negative Memory

- Dead-end memory type.
- Positive and negative anchors.
- Alerts before repeated failure patterns.
- Feedback labels for memory usefulness.

### Phase 5: Team and Server Mode

- Optional PostgreSQL adapter.
- HTTP service mode.
- Import/export.
- ACLs, audit, and multi-user governance.

## Repository Guide

| Path | Purpose |
| --- | --- |
| `service.go` | Main embedded service API. |
| `types.go` | Public request/result types. |
| `memory.go` | Memory retrieval and profile behavior. |
| `memory_tools.go` | Generic MCP-style memory tools. |
| `goncho_public_tools.go` | Public agent-facing tool surface. |
| `review.go` / `review_tool.go` | Review queues and review tool behavior. |
| `file_import.go` / `quarantine.go` | Local imports and prompt-injection quarantine. |
| `code_claim_verification.go` | Live verification for remembered code/file claims. |
| `drift_anchor.go` | Negative-memory drift warning logic. |
| `host_integration.go` | Host-facing compatibility metadata. |
| `docs/opensource-memory-systems/` | Research corpus and metaanalysis. |
| `docs-site/` | Starlight documentation site. |
| `http/` | HTTP-facing routes and tests. |
| `toolmeta/` | Tool metadata helpers. |

## Documentation Site

The Starlight documentation site lives in [`docs-site/`](docs-site/). The default static deployment target is `https://trebuchetdynamics.github.io/goncho/`.

Operator and integration entry points:

- [Operator Runbook](docs-site/src/content/docs/operators/runbook.md) — deployment, backup, health checks, release checks, and troubleshooting.
- [Gormes Agent Integration](docs-site/src/content/docs/integrations/gormes-agent.md) — recommended seam for plugging Goncho into a Gormes-style Go agent host.
- [`integration/gormes`](integration/gormes) — import-ready adapter that opens SQLite, runs migrations, creates `goncho.Service`, wires public tools, reports status, and closes cleanly.

```bash
cd docs-site
npm install
npm run dev
```

For CI and deploy builds:

```bash
cd docs-site
npm ci
npm run build
```

The repository workflow builds docs on pull requests and publishes `docs-site/dist` from `main` when GitHub Pages is configured to use GitHub Actions as its source.

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

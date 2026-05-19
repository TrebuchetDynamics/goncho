# Goncho

**Local-first context and belief memory for Go agents.**

Goncho is a Go library for agents that need durable memory without turning memory into a cloud service, a vector database project, or a pile of prompt-stuffed notes.

It runs inside your Go process, stores state in SQLite, and exposes Honcho-compatible primitives for profile, search, context, chat, and conclusions.

> Goncho is not trying to be the biggest memory store. It is trying to help agents know what they know, why they know it, when it may be stale, and what not to repeat.

```bash
go get github.com/TrebuchetDynamics/goncho
```

[![Go Reference](https://pkg.go.dev/badge/github.com/TrebuchetDynamics/goncho.svg)](https://pkg.go.dev/github.com/TrebuchetDynamics/goncho)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

---

## Documentation Site

The Starlight documentation site lives in [`docs-site/`](docs-site/). The default static deployment target is `https://trebuchetdynamics.github.io/goncho/`.

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

## Why Goncho Exists

Most agent memory systems start as retrieval systems:

```text
old text -> vector search -> top-k chunks -> prompt
```

That works until the agent has to handle real engineering memory:

- A fact used to be true but is stale now.
- A user preference changed last week.
- A file moved, but old memory still mentions the old path.
- A previous fix failed three times and should not be repeated.
- A memory came from an untrusted import and should not steer behavior.
- A cloud memory service is unacceptable for private developer workflows.

Goncho treats memory as **trust-preserving context architecture**:

```text
raw evidence
  -> claims
  -> scoped temporal beliefs
  -> task-specific orientation
  -> agent action
  -> consolidation, revision, or forgetting
```

Vectors are useful. Search is useful. But they are not the source of truth. Goncho's source of truth is local, auditable memory with scope, provenance, time, and lifecycle state.

## What You Get

| Capability | What it means |
| --- | --- |
| **Local-first storage** | SQLite by default. No required sidecar, hosted API, or vector database. |
| **Honcho compatibility** | Preserve familiar `honcho_profile`, `honcho_search`, `honcho_context`, `honcho_chat`, and `honcho_conclude` semantics. |
| **MCP memory tools** | Includes `store_memory`, `retrieve_memory`, `update_memory`, `summarize_memories`, and `forget_memory` contracts. |
| **Scoped memory** | Model memory by peer, session, workspace, project, or agent boundary. |
| **Temporal beliefs** | Keep track of what was observed, what is current, and what may be stale. |
| **Context packs** | Build compact prompt-ready orientation instead of dumping every matching memory. |
| **Evidence and claims** | Preserve raw observations separately from interpreted conclusions. |
| **Negative memory** | Remember dead ends, failed attempts, and patterns the agent should avoid. |
| **Optional adapters** | LLMs and embedders can improve summaries/search, but the base system works without them. |

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

## The Core Model

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
│  sessions    decisions        validity        with citations  │
│                                                              │
└───────────────────────────────┬──────────────────────────────┘
                                │
                                ▼
                         SQLite database
```

### Evidence

Raw observations from sessions, tools, imports, files, and user messages. Evidence should be preserved before interpretation.

### Claims

Interpreted facts derived from evidence: preferences, decisions, project facts, summaries, warnings, and conclusions.

### Scoped beliefs

Claims become useful only when they have scope and time: who they apply to, which project they belong to, when they were observed, and whether newer evidence supersedes them.

### Orientation packs

Agents do not need a memory dump. They need a small working set:

- current peer or project profile,
- relevant canonical facts,
- recent high-signal episodes,
- known dead ends,
- unresolved conflicts,
- verification warnings,
- citations back to source memory.

## Core API

Four service calls cover the most common embedded use case:

```go
svc := goncho.NewService(db, cfg, log)

// Who is this peer?
profile, err := svc.Profile(ctx, "telegram:12345")

// Remember durable profile facts.
err = svc.SetProfile(ctx, "telegram:12345", []string{
    "Go developer",
    "Prefers local-first tools",
})

// Search what Goncho knows.
results, err := svc.Search(ctx, goncho.SearchParams{
    Query: "database preferences",
    Limit: 5,
})

// Build prompt-ready context.
pack, err := svc.Context(ctx, goncho.ContextParams{
    Peer:      "telegram:12345",
    MaxTokens: 8000,
})
```

Full API reference: [pkg.go.dev/github.com/TrebuchetDynamics/goncho](https://pkg.go.dev/github.com/TrebuchetDynamics/goncho)

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

MCP memory contracts are available for hosts that use generic memory tools:

```text
store_memory
retrieve_memory
update_memory
summarize_memories
forget_memory
```

This lets a host start with simple memory tools and grow into richer context packs without changing the agent-facing vocabulary.

Migration guide → [docs-site/src/content/docs/reference/honcho-compatibility.md](docs-site/src/content/docs/reference/honcho-compatibility.md)

## Design Principles

Goncho follows seven principles from the project research in [`docs/opensource-memory-systems/METAANALYSIS-MEMORY-SYSTEMS.md`](docs/opensource-memory-systems/METAANALYSIS-MEMORY-SYSTEMS.md):

1. **Evidence before memory** — preserve raw events and tool outputs first; derive memories second.
2. **Claims, not chunks** — store what is believed with proof, confidence, scope, and time.
3. **Hooks over manual saves** — capture at cognitive transition points such as session start, tool use, compaction, and stop.
4. **Orientation, not dumping** — inject compact task context, not every semantically similar memory.
5. **Negative memory matters** — failed paths and rejected approaches are part of intelligence.
6. **Small agent surface** — expose stable primitives for context, search, remember, review, and handoff.
7. **Trust is the moat** — every surfaced memory should answer: why this, why now, and why trust it?

## Comparison

| Approach | Strength | Failure mode Goncho avoids |
| --- | --- | --- |
| Flat markdown memory | Simple and editable | Becomes token bloat without search, scope, or lifecycle. |
| Vector-only memory | Good fuzzy recall | Returns plausible but stale or wrong chunks. |
| Cloud memory APIs | Easy hosted setup | Adds network dependency, privacy risk, and vendor lock-in. |
| Postgres + pgvector stacks | Powerful production search | Raises the setup floor for a single local Go agent. |
| Goncho | Local evidence, scoped beliefs, compact context | Keeps the base workflow embedded, auditable, and offline-capable. |

## Current Status

Goncho is pre-release software. The repository is actively building the local memory kernel and compatibility layer.

| Area | Status |
| --- | --- |
| Embedded Go service | In progress |
| SQLite storage and migrations | In progress |
| Peer profiles and conclusions | In progress |
| Search and context APIs | In progress |
| Honcho-compatible tool names | In progress |
| MCP memory tool contracts | In progress |
| Local markdown/import workflows | Experimental |
| Lifecycle stewardship and review queues | Experimental |
| Graph, drift, and dashboard layers | Planned |

## Roadmap

### Phase 1: Local Memory Kernel

- SQLite storage.
- FTS-backed search.
- Peer profiles, conclusions, summaries, and context packs.
- Honcho-compatible service and tool names.
- MCP memory contracts.
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
| `host_integration.go` | Host-facing compatibility metadata. |
| `docs/opensource-memory-systems/` | Research corpus and metaanalysis. |
| `docs/superpowers/` | Design specs and implementation plans. |
| `http/` | HTTP-facing routes and tests. |
| `toolmeta/` | Tool metadata helpers. |

## Development

```bash
git clone https://github.com/TrebuchetDynamics/goncho.git
cd goncho
go test ./...
```

If tests fail, treat the first compiler or test error as the source of truth. This repository intentionally favors evidence over optimistic status claims.

## License

MIT

---

Goncho is developed by [Trebuchet Dynamics](https://github.com/TrebuchetDynamics) as part of a local-first agent infrastructure ecosystem.

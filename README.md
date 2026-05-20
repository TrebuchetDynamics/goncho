# Goncho

**High-trust memory runtime for Go agents.**

Goncho gives agent hosts durable local memory with auditability, scoped recall, review warnings, and live-verification discipline. It is not a hosted memory API, a vector database wrapper, or a giant tool catalog.

Goncho's rule is simple:

```text
evidence before belief
live verification before action
```

If memory says a file exists, verify it. If memory says a migration was approved, verify it. If memory says an API path still exists, verify it. Goncho treats memory as orientation until current evidence says it is safe to act.

```bash
go get github.com/TrebuchetDynamics/goncho
```

[![Go Reference](https://pkg.go.dev/badge/github.com/TrebuchetDynamics/goncho.svg)](https://pkg.go.dev/github.com/TrebuchetDynamics/goncho)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

---

## Why Goncho

Most agent memory systems optimize for breadth: more connectors, more tools, more autonomous behavior. Goncho optimizes for trust: memory correctness, bounded writes, reproducible retrieval, local state, and verification before action.

Goncho is inspired by broad integration systems like [`agentmemory`](docs/opensource-memory-systems/agentmemory/README.md), but it makes a different product bet:

```text
agentmemory: broad integration layer
Goncho:      high-trust memory runtime
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

Benchmarks:

```bash
go test ./cmd/goncho-bench
make bench-longmemeval-s-smoke
```

Retrieval benchmark docs: [docs-site/src/content/docs/reference/retrieval-benchmarks.md](docs-site/src/content/docs/reference/retrieval-benchmarks.md)

---

## Documentation

Start here:

- [Current Capabilities](docs-site/src/content/docs/start/current-capabilities.md)
- [Quick Start](docs-site/src/content/docs/start/quick-start.md)
- [Core API](docs-site/src/content/docs/reference/core-api.md)
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

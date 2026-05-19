# Goncho

**Persistent agent memory without infrastructure headaches.**

Most agent-memory systems become distributed systems before your agent even remembers a preference. You need a Postgres instance, a vector database, a sidecar process, and three API keys — just to remember that a user prefers dark mode.

Goncho runs entirely in your Go binary. One import. One SQLite file. Zero external dependencies.

```
go get github.com/TrebuchetDynamics/goncho
```

[![Go Reference](https://pkg.go.dev/badge/github.com/TrebuchetDynamics/goncho.svg)](https://pkg.go.dev/github.com/TrebuchetDynamics/goncho)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

---

## The Problem

Your agent talks to a user. The user says *"I work in finance, prefer concise answers, and use Telegram."*

Next session, the agent asks: *"What do you do for work?"*

This is the agent memory problem. The solutions on the market are:

| Approach | What You Actually Get |
|----------|----------------------|
| **Hosted memory (mem0, Zep)** | Cloud dependency, API keys, vendor lock-in, $50/mo to remember preferences |
| **Vector DB + embeddings** | Postgres + pgvector or Pinecone + OpenAI embeddings + a sidecar service |
| **Raw text files** | Works until you need search, dedup, or multi-peer isolation |
| **Nothing** | Your agent has amnesia every session |

Goncho is the fourth option: **a memory system that ships as a Go library and works offline from line one.**

## Quick Start

```go
package main

import (
    "context"
    "database/sql"
    "fmt"

    _ "github.com/ncruces/go-sqlite3/driver"
    "github.com/TrebuchetDynamics/goncho"
)

func main() {
    db, _ := sql.Open("sqlite3", "memory.db")
    defer db.Close()

    goncho.RunMigrations(db)

    svc := goncho.NewService(db, goncho.Config{
        WorkspaceID: "my-agent",
        Observer:    "assistant",
    }, nil)

    ctx := context.Background()

    // Session 1: user tells you about themselves
    svc.SetProfile(ctx, "telegram:12345", []string{
        "Works in finance",
        "Prefers concise answers",
        "Uses Telegram",
    })

    // Session 2 (next day, new process, same DB):
    card, _ := svc.Profile(ctx, "telegram:12345")
    fmt.Println(card.Card)
    // [Works in finance Prefers concise answers Uses Telegram]
}
```

Same database. Same file. No restart. No cloud call. No embeddings required.

## What It Remembers

Goncho models memory as composable artifacts, each with a specific role:

| Artifact | What It Is | Example |
|----------|-----------|---------|
| **Peer Card** | Grounding facts about a peer | *"User is a Go developer, prefers SQLite"* |
| **Conclusion** | Derived or authored facts | *"User abandoned Postgres after migration pain"* |
| **Summary** | Compressed session history | Short (every 20 msgs), long (every 60 msgs) |
| **Context** | Token-budgeted read product | Assembled from the above + recent messages |

### Before Goncho

```
User: I prefer SQLite over Postgres.
Assistant: Got it!

--- next session ---

Assistant: What database do you use?
User: ...we just talked about this.
```

### After Goncho

```
User: I prefer SQLite over Postgres.
Assistant: [stores conclusion via goncho.Conclude()]

--- next session, new process, same memory.db ---

Assistant: [goncho.Context() assembles peer card + conclusions]
Assistant: Since you prefer SQLite, here's a schema design...
User: Yes, exactly what I needed.
```

## Local-First Markdown

Goncho can back memory with plain markdown files you can open in any editor:

```markdown
# Goncho Memory

## Entry: user-preference-database
- **Agent:** assistant
- **Peer:** telegram:12345
- **Scope:** workspace
- **Created:** 2026-05-19

User prefers SQLite over Postgres after a painful migration
experience with pgvector. Values simplicity over feature richness.

---

## Entry: user-work-context
- **Agent:** assistant
- **Peer:** telegram:12345
- **Scope:** workspace
- **Created:** 2026-05-19

Works in finance. Prefers concise, direct answers without
explanation unless asked. Uses Telegram as primary platform.
```

Edit the file. Goncho detects the change on next read. Conflict-aware. Restart-persistent. No cloud API.

## Comparison

| | Goncho | mem0/Zep | Postgres + pgvector | Raw files |
|---|---|---|---|---|
| **Deployment** | `go get` | Cloud or Docker | Docker + managed DB | Manual |
| **Storage** | SQLite — one file | Managed service | Postgres cluster | Text files |
| **API Keys** | None | Required | Required (embeddings) | None |
| **Search** | FTS + filter grammar | Vector similarity | Vector + FTS | `grep` |
| **Multi-peer** | Workspace + peer scoped | Account-scoped | Schema-scoped | None |
| **Editable** | Markdown or SQLite | Dashboard only | SQL only | Yes |
| **Offline** | Full | Partial | Partial | Full |
| **Honcho compat** | Drop-in | No | No | No |

## Core API

Five methods cover 90% of use cases:

```go
svc := goncho.NewService(db, cfg, log)

// Who is this peer?
card, _ := svc.Profile(ctx, "telegram:12345")

// Remember something about them
svc.SetProfile(ctx, "telegram:12345", []string{"Go developer", "prefers SQLite"})

// Search what you know
results, _ := svc.Search(ctx, goncho.SearchParams{Query: "database preferences"})

// Build context for prompt injection
context, _ := svc.Context(ctx, goncho.ContextParams{
    Peer:      "telegram:12345",
    MaxTokens: 8000,
})

// Store a durable conclusion
svc.Conclude(ctx, goncho.ConcludeParams{
    Peer:    "telegram:12345",
    Content: "Abandoned Postgres after migration pain",
})
```

Full API reference → [pkg.go.dev](https://pkg.go.dev/github.com/TrebuchetDynamics/goncho)

## Optional Adapters

Goncho works with zero external dependencies. Two optional adapters unlock enhanced capabilities:

```go
type LLM interface {
    Generate(ctx context.Context, prompt string, opts ...GenerateOption) (string, error)
}

type Embedder interface {
    Embed(ctx context.Context, texts []string) ([][]float64, error)
}
```

| Adapter | Without It | With It |
|---------|-----------|---------|
| **LLM** | Manual conclusions only | Auto-generated summaries, reasoning |
| **Embedder** | FTS-only search | Vector-backed semantic search |

Both are nil-safe. Pass `nil`, Goncho degrades gracefully. No startup failures.

## MCP & Honcho Compatibility

Goncho exposes MCP-compatible memory tools and maintains full Honcho v3 tool-name compatibility:

| Goncho Tool | Honcho Alias | Purpose |
|-------------|-------------|---------|
| `store_memory` | — | Persist a memory entry |
| `retrieve_memory` | — | Search memories |
| — | `honcho_profile` | Read/write peer cards |
| — | `honcho_search` | Search conclusions |
| — | `honcho_context` | Assemble prompt context |
| — | `honcho_chat` | Dialectic peer chat |
| — | `honcho_conclude` | Create manual conclusions |

Drop-in replacement for any Honcho integration. See [docs/05-from-honcho.md](docs/05-from-honcho.md) for the migration guide.

## Architecture

```
┌─────────────────────────────────────────────────┐
│                   Your Go Binary                 │
│                                                   │
│  ┌──────────┐    ┌───────────┐    ┌───────────┐  │
│  │  Kernel  │───▶│  Goncho   │───▶│  SQLite   │  │
│  │          │    │  Service  │    │  (single  │  │
│  │  Agent   │◀───│           │◀───│   file)   │  │
│  │  Loop    │    │           │    │           │  │
│  └──────────┘    └───────────┘    └───────────┘  │
│                       │                           │
│              ┌────────┴────────┐                  │
│              │  Optional:      │                  │
│              │  LLM Adapter    │                  │
│              │  Embedder       │                  │
│              └─────────────────┘                  │
└─────────────────────────────────────────────────┘
```

No HTTP loopback. No extra process. No RPC bridge. No cloud dependency.

## Status

**v0.1.0** — Pre-release. The public API is stable enough for integration but may change before v1.0.0.

| Capability | Status |
|------------|--------|
| Peer cards, conclusions, context assembly | ✅ |
| Session summaries (short/long cadence) | ✅ |
| Search (FTS + filter grammar) | ✅ |
| MCP memory tools + Honcho compatibility | ✅ |
| Local-first markdown store | ✅ |
| Dialectic chat contract | ✅ |
| Operator diagnostics (`goncho doctor`) | ✅ |
| Memory V1 compatibility contract | ✅ |
| Workspace isolation (global scope) | 📋 |

## License

MIT

---

*Goncho is developed as part of the [Trebuchet Dynamics](https://github.com/TrebuchetDynamics) agent infrastructure ecosystem.*

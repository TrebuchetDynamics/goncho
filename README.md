# Goncho

**Honcho-compatible, local-first memory system for Go agents.**

> Your memory layer should not require cloud infrastructure just to remember who the user is.

Goncho runs entirely in your Go binary — no sidecar, no cloud dependency, no mandatory API key. One import. One SQLite file. Your agent remembers across sessions without becoming a distributed system.

```
go get github.com/TrebuchetDynamics/goncho
```

[![Go Reference](https://pkg.go.dev/badge/github.com/TrebuchetDynamics/goncho.svg)](https://pkg.go.dev/github.com/TrebuchetDynamics/goncho)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

---

## Why This Exists

Most agent-memory stacks require Postgres, a vector database, and three API keys before your agent can remember a single user preference. That's not memory — that's infrastructure debt.

| What You're Comparing Against | What You Actually Get |
|-------------------------------|----------------------|
| **mem0, Zep, langmem** | Cloud dependency, API keys, $50/mo, vendor lock-in, network latency on every recall |
| **Postgres + pgvector** | Docker compose file, connection pooling, embedding pipeline, OpenAI bill |
| **Raw markdown files** | Works until you need search, dedup, or multi-peer isolation |
| **Nothing** | Your agent has amnesia every session |

Goncho is the fourth option: **a memory system that ships as a Go library and works offline from line one.**

## Memory That Persists

### Session 1 — Tuesday, 2pm

```
User: I prefer SQLite over Postgres. Had a bad experience
      migrating to pgvector last year. Also, I work in finance.

Assistant: Got it. [goncho stores this automatically]
```

### Session 2 — Wednesday, 9am (new process, same machine)

```
Assistant: [goncho.Context() assembles the peer card + conclusions]
Assistant: Since you prefer SQLite, here's a schema design that
           avoids the pgvector migration issues you ran into.

User: Yes, exactly what I needed.
```

Same `memory.db` file. No cloud call. No embeddings. No restart ceremony. The agent just *remembers*.

## Editable Memory

Goncho can back memory with plain markdown files. Open them in any editor. Edit them by hand. Goncho detects changes on next read.

```markdown
# Goncho Memory

## user-preference-database
- **Peer:** telegram:12345
- **Created:** 2026-05-19

User prefers SQLite over Postgres after a painful migration
experience with pgvector. Values simplicity over feature richness.

## user-work-context
- **Peer:** telegram:12345
- **Created:** 2026-05-19

Works in finance. Prefers concise, direct answers without
explanation unless asked. Uses Telegram as primary platform.
```

Change a fact in your editor. Next session, the agent sees it. No API. No dashboard. Just a file.

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

    // Store what you learn about a peer
    svc.SetProfile(ctx, "telegram:12345", []string{
        "Works in finance",
        "Prefers SQLite over Postgres",
    })

    // Next session, it's still there
    card, _ := svc.Profile(ctx, "telegram:12345")
    fmt.Println(card.Card)
}
```

## What It Remembers

| Artifact | Purpose | Example |
|----------|---------|---------|
| **Peer Card** | Grounding facts about a peer | *"User is a Go developer, prefers SQLite"* |
| **Conclusion** | Derived or authored facts | *"Abandoned Postgres after migration pain"* |
| **Summary** | Compressed session history | Short (every 20 msgs), long (every 60 msgs) |
| **Context** | Token-budgeted read product | Card + conclusions + summary + recent messages |

## Core API

Four methods cover most use cases:

```go
svc := goncho.NewService(db, cfg, log)

// Who is this peer?
card, _ := svc.Profile(ctx, "telegram:12345")

// Remember something
svc.SetProfile(ctx, "telegram:12345", []string{"Go developer", "prefers SQLite"})

// Search what you know
results, _ := svc.Search(ctx, goncho.SearchParams{
    Query: "database preferences",
})

// Build context for prompt injection
ctx, _ := svc.Context(ctx, goncho.ContextParams{
    Peer:      "telegram:12345",
    MaxTokens: 8000,
})
```

Full API → [pkg.go.dev](https://pkg.go.dev/github.com/TrebuchetDynamics/goncho)

## Zero External Dependencies

Goncho works without LLMs or embeddings. Both are optional adapters:

| | Without | With |
|---|---------|------|
| **LLM adapter** | Manual conclusions only | Auto-generated summaries, reasoning |
| **Embedder adapter** | FTS-only search | Vector-backed semantic search |

Pass `nil` for either. Goncho degrades gracefully — no startup failures, no missing-feature errors.

## Honcho Compatibility

Drop-in replacement for any Honcho integration. Tool names stay `honcho_profile`, `honcho_search`, `honcho_context`, `honcho_chat`, `honcho_conclude`. MCP memory tools (`store_memory`, `retrieve_memory`, etc.) included.

Migration guide → [docs/05-from-honcho.md](docs/05-from-honcho.md)

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
│              │  LLM / Embedder │                  │
│              └─────────────────┘                  │
└─────────────────────────────────────────────────┘
```

No HTTP loopback. No extra process. No RPC bridge. No cloud dependency.

## Status

**v0.1.0** — Pre-release. The public API is stable enough for integration.

| | |
|---|---|
| Peer cards, conclusions, context | ✅ Stable |
| Search with filter grammar | ✅ Stable |
| Session summaries | ✅ Stable |
| Local-first markdown | ✅ Stable |
| Honcho + MCP compatibility | ✅ Stable |
| Migration-safe (V1 contract frozen) | ✅ Stable |

## License

MIT

---

*Goncho is developed as part of the [Trebuchet Dynamics](https://github.com/TrebuchetDynamics) agent infrastructure ecosystem.*

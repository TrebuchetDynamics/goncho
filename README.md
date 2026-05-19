# Goncho

**Honcho-compatible, local-first memory system for Go agents.**

Goncho embeds durable, peer-scoped agent memory directly into your Go binary — no sidecar, no cloud dependency, no mandatory API key. It is the Go-native implementation of the [Honcho](https://github.com/gethoncho/honcho) memory contract, designed for agents that need persistent context across sessions without external infrastructure.

```
go get github.com/TrebuchetDynamics/goncho
```

[![Go Reference](https://pkg.go.dev/badge/github.com/TrebuchetDynamics/goncho.svg)](https://pkg.go.dev/github.com/TrebuchetDynamics/goncho)
[![Go Report Card](https://goreportcard.com/badge/github.com/TrebuchetDynamics/goncho)](https://goreportcard.com/report/github.com/TrebuchetDynamics/goncho)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

---

## Why Goncho

Most agent memory systems require a hosted service, a vector database, or cloud credentials. Goncho runs entirely in-process:

| Feature | Goncho | Typical Alternatives |
|---------|--------|---------------------|
| **Deployment** | `go get` — single binary | Docker containers, managed services |
| **Storage** | SQLite — one file on disk | Postgres, Pinecone, LanceDB, mem0 |
| **API Keys** | None required | OpenAI, cloud vector DBs, hosted memory |
| **Compatibility** | Honcho v3 tool names & contracts | Proprietary APIs |
| **Offline** | Fully functional | Requires network |
| **Editable** | Plain markdown memory files | Opaque binary stores |

## Quick Start

```go
package main

import (
    "context"
    "database/sql"
    "fmt"
    "log/slog"
    "os"

    _ "github.com/ncruces/go-sqlite3/driver"
    "github.com/TrebuchetDynamics/goncho"
)

func main() {
    db, err := sql.Open("sqlite3", "memory.db")
    if err != nil {
        panic(err)
    }
    defer db.Close()

    // Run schema migrations
    if err := goncho.RunMigrations(db); err != nil {
        panic(err)
    }

    // Create the service
    svc := goncho.NewService(db, goncho.Config{
        WorkspaceID: "my-workspace",
        Observer:    "my-agent",
    }, slog.New(slog.NewTextHandler(os.Stdout, nil)))

    ctx := context.Background()

    // Store a peer card
    if err := svc.SetProfile(ctx, "telegram:12345", []string{
        "User is a Go developer",
        "Prefers SQLite over Postgres",
        "Works on agent infrastructure",
    }); err != nil {
        panic(err)
    }

    // Retrieve the peer card
    result, err := svc.Profile(ctx, "telegram:12345")
    if err != nil {
        panic(err)
    }
    fmt.Printf("Peer card: %v\n", result.Card)

    // Search conclusions
    searchResult, err := svc.Search(ctx, goncho.SearchParams{
        Query: "Go developer preferences",
    })
    fmt.Printf("Found %d results\n", len(searchResult.Conclusions))
}
```

## Core Concepts

### Artifacts

Goncho models memory as composable artifacts, each with a specific role:

| Artifact | Scope | Purpose |
|----------|-------|---------|
| **Peer Card** | Global per peer | Compact grounding facts about a peer (user, agent, system) |
| **Conclusion** | Workspace + peer + session | Durable derived or operator-authored facts |
| **Summary** | Per session | Compressed history — short (every 20 msgs) and long (every 60 msgs) |
| **Representation** | Derived on read | Perspective-sensitive view of a peer's knowledge state |
| **Context** | Token-budgeted read | Assembled product: card + representation + conclusions + summary + recent messages |

### Identity Mapping

```
workspace:  "my-workspace"          ← agent namespace
ai peer:    "my-agent"              ← the agent's identity
user peer:  "telegram:12345"        ← stable platform identity
session:    "chat-abc-123"          ← conversation boundary
```

### Read Path

When an agent needs context, Goncho assembles it deterministically:

1. **Peer card** — who is this peer?
2. **Representation** — what does this peer know, from whose perspective?
3. **Conclusions** — what facts have been derived about this peer?
4. **Summary** — what happened in this session recently?
5. **Recent messages** — the last N turns, within token budget

### Write Path

Goncho's write path is cheap and non-blocking:

1. Kernel persists raw turn
2. Goncho projector resolves workspace, session, and peer identity
3. Worker batches turns by session scope
4. Extractor emits observations and evidence links
5. Deriver consolidates conclusions with dedupe
6. Summary scheduler produces short and long summaries
7. Caches are invalidated or refreshed

## Service API

```go
// Service is the main facade. All methods are safe for concurrent use.
type Service struct { /* ... */ }

func NewService(db *sql.DB, cfg Config, log *slog.Logger) *Service

// Profile reads or updates a peer's grounding card.
func (s *Service) Profile(ctx context.Context, peer string) (ProfileResult, error)
func (s *Service) SetProfile(ctx context.Context, peer string, card []string) error

// Search finds conclusions matching a query, with optional filters.
func (s *Service) Search(ctx context.Context, params SearchParams) (SearchResultSet, error)

// Context assembles a token-budgeted read product for prompt injection.
func (s *Service) Context(ctx context.Context, params ContextParams) (ContextResult, error)

// Conclude creates durable manual conclusions.
func (s *Service) Conclude(ctx context.Context, params ConcludeParams) (ConcludeResult, error)

// Chat provides Honcho-compatible peer.chat with reasoning levels.
func (s *Service) Chat(ctx context.Context, peer string, params ChatParams) (ChatResult, error)

// CreateMessages stores a batch of session messages.
func (s *Service) CreateMessages(ctx context.Context, params CreateMessagesParams) (CreateMessagesResult, error)

// DeleteSession cascades session-scoped data while preserving peer cards.
func (s *Service) DeleteSession(ctx context.Context, sessionKey string) (SessionDeletionResult, error)

// DeleteWorkspace cascades all data for one workspace.
func (s *Service) DeleteWorkspace(ctx context.Context) (WorkspaceDeletionResult, error)
```

### Injectable Adapters

Goncho's `Service` accepts optional adapters for enhanced capabilities:

```go
type LLM interface {
    Generate(ctx context.Context, prompt string, opts ...GenerateOption) (string, error)
}

type Embedder interface {
    Embed(ctx context.Context, texts []string) ([][]float64, error)
}
```

When `LLM` is nil, reasoning and summary generation return degraded-mode results. When `Embedder` is nil, search falls back to FTS-only. This makes Goncho fully functional with zero external dependencies.

## MCP Tools

Goncho exposes MCP-compatible memory tools out of the box:

| Tool | Description |
|------|-------------|
| `store_memory` | Persist a memory entry with metadata |
| `retrieve_memory` | Search and retrieve memories |
| `update_memory` | Modify an existing memory |
| `summarize_memories` | Generate a summary of multiple memories |
| `forget_memory` | Soft-delete (tombstone) a memory |

All tools also work under their Honcho-compatible names (`honcho_profile`, `honcho_search`, `honcho_context`, `honcho_chat`, `honcho_conclude`, `honcho_reasoning`) for drop-in compatibility with existing Honcho integrations.

## Local-First Markdown

Goncho supports a local-first memory mode where memories are stored as plain markdown files:

- **User-readable/editable** — open `GONCHO_MEMORY.md` in any editor
- **Restart-persistent** — survives process and machine restarts
- **Conflict-aware** — detects and reports edit conflicts between file and SQLite
- **MCP-compatible** — any agent framework can access via MCP tools
- **No cloud dependency** — works fully offline

## CLI

```bash
# Install
go install github.com/TrebuchetDynamics/goncho/cmd/goncho@latest

# Run diagnostics
goncho doctor --db memory.db

# Search memories
goncho search --query "user preferences" --db memory.db

# Inspect memory state
goncho memory status --db memory.db

# Replay retrieval traces
goncho recall-replay --db memory.db
```

## Honcho Compatibility

Goncho implements the Honcho v3 memory contract. If you're migrating from Honcho:

- **Tool names** — `honcho_*` tools work identically
- **Filter grammar** — AND, OR, NOT, gt, gte, lt, lte, ne, in, contains, icontains, metadata, wildcard
- **Peer cards** — max 40 facts, directional (observer/observed), manual replacement
- **Session summaries** — short/long cadence at 20/60 message intervals
- **Context assembly** — token-budgeted with configurable allocation
- **Dialectic chat** — reasoning levels (low/medium/high), degraded streaming support

See [docs/05-from-honcho.md](docs/05-from-honcho.md) for the full migration guide.

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

**v0.1.0** — Pre-release. The public API (`Service` methods, `Config`, all param/result types) is stable enough for integration but may change before v1.0.0.

| Area | Status |
|------|--------|
| Peer cards (directional, max-40) | ✅ Complete |
| Conclusions (FTS, filters, dedupe) | ✅ Complete |
| Context assembly (token budget) | ✅ Complete |
| Session summaries (short/long) | ✅ Complete |
| Search (FTS + filter grammar) | ✅ Complete |
| Dialectic chat contract | ✅ Complete |
| CRUD lifecycle invariants | ✅ Complete |
| Dream scheduler (work intent) | ✅ Complete |
| MCP memory tools | ✅ Complete |
| Local-first markdown store | ✅ Complete |
| Webhook delivery worker | ✅ Complete |
| JWT scoped keys | ✅ Complete |
| Dynamic agent registry | ✅ Complete |
| Streaming chat persistence | ✅ Complete |
| Operator diagnostics (doctor) | ✅ Complete |
| Memory V1 compatibility contract | ✅ Complete |
| Workspace isolation (global scope) | 📋 Planned |

## Project Structure

```
goncho/
├── service.go              # Main Service facade
├── types.go                # Config, params, result types
├── store_sqlite.go         # SQLite storage layer
├── memory_tools.go         # MCP memory tool implementations
├── store_markdown.go       # Local-first markdown store
├── dream.go                # Dream scheduling
├── filter.go               # Search filter grammar
├── keys.go                 # JWT scoped keys
├── webhooks.go             # Webhook endpoint CRUD
├── webhook_delivery.go     # Webhook delivery worker
├── dynamic_agents.go       # Dynamic agent registry
├── docs/                   # User-facing documentation
├── examples/               # Runnable example programs
└── reference/              # Upstream Honcho parity fixtures
```

## License

MIT

---

*Goncho is developed as part of the [Trebuchet Dynamics](https://github.com/TrebuchetDynamics) agent infrastructure ecosystem.*

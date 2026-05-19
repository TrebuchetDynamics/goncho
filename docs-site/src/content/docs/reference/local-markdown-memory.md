---
title: Local Markdown Memory
description: Human-editable memory backed by SQLite and markdown.
---

Goncho's local markdown memory store persists tool memory to SQLite and mirrors it to a markdown file.

Current exported constructor and config:

- `NewLocalMarkdownMemoryStore`
- `LocalMarkdownMemoryConfig`

```go
store := goncho.NewLocalMarkdownMemoryStore(db, goncho.LocalMarkdownMemoryConfig{
	Path:           "memory.md",
	AgentID:        "assistant",
	WorkspaceID:    "my-agent",
	ObserverPeerID: "assistant",
	PeerID:         "telegram:12345",
	SessionID:      "session-1",
})
```

The store reports a local-first status, reloads markdown before retrieval, and exports after writes.

## Why It Matters

Editable markdown is a repair surface. Humans can inspect and correct memory without a hosted dashboard or opaque vector index.

:::caution[Operational boundary]
Treat markdown memory files as application data. Protect and back them up according to the host application's policy.
:::

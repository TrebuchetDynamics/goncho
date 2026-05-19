---
title: Memory Tools
description: Generic agent-facing memory tools exposed by Goncho.
---

Goncho exposes generic memory tools around [`MemoryToolStore`](https://pkg.go.dev/github.com/TrebuchetDynamics/goncho#MemoryToolStore).

| Tool | Purpose |
| --- | --- |
| `store_memory` | Persist information to agent memory. |
| `retrieve_memory` | Retrieve memories relevant to a query. |
| `update_memory` | Correct content or adjust importance. |
| `summarize_memories` | Summarize related memories by query or tag. |
| `forget_memory` | Soft-delete an active memory entry. |

These tools are a small integration surface, not the whole memory model. Internally, Goncho can preserve richer state than it exposes to an agent prompt.

:::note[Current status]
The generic memory tools are local-first and markdown-backed when used with `LocalMarkdownMemoryStore`.
:::

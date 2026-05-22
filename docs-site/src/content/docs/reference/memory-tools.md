---
title: Memory Tools
description: Generic agent-facing memory tools exposed by Goncho.
---

Goncho exposes generic memory tools around `MemoryToolStore`.

| Tool | Purpose |
| --- | --- |
| `store_memory` | Persist information to agent memory. |
| `retrieve_memory` | Retrieve memories relevant to a query. |
| `update_memory` | Correct content or adjust importance. |
| `summarize_memories` | Summarize related memories by query or tag. |
| `forget_memory` | Soft-delete an active memory entry. |
| `goncho_review` | List and resolve conflict/stale review items, with enum-validated `status`/`kind` filters plus `subject_id` and `related_id` filters for inspecting review or supersession chains. |

These tools are a small integration surface, not the whole memory model. Internally, Goncho can preserve richer state than it exposes to an agent prompt.

:::note[Current status]
The generic memory tools are local-first and markdown-backed when used with `LocalMarkdownMemoryStore`.
:::

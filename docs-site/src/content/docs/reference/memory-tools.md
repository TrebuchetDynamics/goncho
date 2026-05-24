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
| `goncho_recall` | Run auditable service recall and return a scored `RecallTrace`, selected/rejected counts, warnings, deterministic replay evidence, and a compact diagnostics report; pass `compact: true` to omit full trace/replay payloads while keeping diagnostics. |
| `goncho_review` | List and resolve conflict/stale review items, with enum-validated `status`/`kind` filters plus `subject_id` and `related_id` filters for inspecting review or supersession chains. |

For `goncho_review` list requests, use `open` or `resolved` for `status` and `conflict` or `stale` for `kind`; invalid values return an error instead of an empty queue. Omitted or blank `status` defaults to open review items. For resolve requests, use `accepted`, `rejected`, `superseded`, or `verified` for `resolution`; invalid values return enum-specific guidance and leave the review item open.

These tools are a small integration surface, not the whole memory model. Internally, Goncho can preserve richer state than it exposes to an agent prompt.

Use this reference with the [Core API](/reference/core-api/) and [Operator Runbook](/operators/runbook/) when choosing which agent-facing tools to register in a host.

:::note[Current status]
The generic memory tools are local-first and markdown-backed when used with `LocalMarkdownMemoryStore`. See [Local Markdown Memory](/reference/local-markdown-memory/) for the editable storage path.
:::

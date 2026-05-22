---
title: Local-First Memory
description: Why Goncho keeps memory in the agent runtime by default.
---

Goncho is designed to run inside a Go binary with local persistence. No hosted memory service is required for the agent to remember across sessions.

## Operational Properties

- Memory can stay in the host process and local filesystem.
- SQLite gives the agent one portable database file.
- [Core API](/reference/core-api/) and [memory tools](/reference/memory-tools/) are the normal integration surfaces for host applications.
- [Local markdown memory](/reference/local-markdown-memory/) gives humans an editable repair surface.
- Optional adapters should improve recall or consolidation without becoming mandatory infrastructure.

## Privacy Boundary

Local-first is not a compliance claim. Treat SQLite databases and markdown memory files as application data: protect, back up, and migrate them according to the host application's policy.

:::caution[Failure mode]
An optional adapter can still send data outside the process. Adapter behavior belongs to the host application's data policy.
:::

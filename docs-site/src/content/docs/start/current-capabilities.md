---
title: Current Capabilities
description: What Goncho supports today and what remains architecture direction.
---

Goncho is usable as a pre-1.0 Go library, with v0.1.x focused on the importable local memory kernel, compatibility surfaces, and deterministic local E2E proof.

| Capability | Today | Direction |
| --- | --- | --- |
| SQLite persistence | `memory.OpenSqlite` initializes service tables; `RunMigrations` initializes observation and audit tables | Clearer operational migration docs and lifecycle guidance |
| Peer cards | `SetProfile`, `Profile`, directional peer-card support | Richer belief provenance |
| Search | `Search` over stored conclusions and fallback session turns | Stronger evidence lineage and ranking diagnostics |
| Context assembly | `Context` returns peer card, conclusions, summaries, search hits, and recent turns | Compact orientation packs with stronger provenance |
| Session summaries | Deterministic short and long summary slots | Hook-native consolidation around lifecycle boundaries |
| Local markdown memory | Shipped local markdown memory store and memory tools | Review and repair workflows around editable memory |
| Honcho compatibility | Honcho-style names and compatibility harnesses exist | Clearer migration boundaries |
| Confidence and staleness | Conceptual direction | First-class scoring contract |
| Temporal validity | Conceptual direction | Scoped validity windows |
| Negative memory review | Conceptual direction | First-class review and repair products |

:::note[Current status]
This site documents both the shipped library and the architecture direction. Pages call out conceptual examples when they describe constraints Goncho has not exposed as stable API.
:::

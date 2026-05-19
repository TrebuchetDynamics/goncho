---
title: Core API
description: The main Goncho service methods and their mental model.
---

Goncho's Go service API is the primary integration surface.

This page names the exported symbols used by the current repository and explains the conceptual surface. Use Go documentation generated from this checkout for exact signatures until public module docs catch up.

| API | Role |
| --- | --- |
| `memory.OpenSqlite` | Open a local SQLite store and initialize the service tables used by peer cards, search, context, summaries, and memory tools. |
| `RunMigrations` | Initialize Goncho observation and audit tables on an opened database. |
| `NewService` | Bind a Goncho service to a database, workspace, and observer. |
| `SetProfile` | Write stable peer-card facts from the default observer perspective. |
| `SetProfileForTarget` | Write a directional representation: one peer's view of another. |
| `Profile` | Read the current peer card. |
| `Search` | Retrieve conclusions or turn evidence relevant to a query. |
| `Context` | Assemble an orientation pack for prompt construction. |
| `Conclude` | Write or delete manual conclusions. |
| `CreateMessages` | Persist session messages as lifecycle evidence. |
| `OnSessionEnd` | Consolidate a completed session into summaries. |

## Directional Representation

Memory is not globally true by default. Goncho can distinguish what one observer believes about a target from what another observer believes.

This prevents one perspective from silently becoming universal truth.

:::note[Current setup note]
For a fresh SQLite database, use `memory.OpenSqlite(...)`, then `goncho.RunMigrations(store.DB())`, then pass `store.DB()` to `goncho.NewService`.
:::

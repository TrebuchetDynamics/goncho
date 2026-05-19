---
title: Core API
description: The main Goncho service methods and their mental model.
---

Goncho's Go service API is the primary integration surface.

Exact signatures live on [pkg.go.dev](https://pkg.go.dev/github.com/TrebuchetDynamics/goncho). This page explains the conceptual surface.

| API | Role |
| --- | --- |
| [`memory.OpenSqlite`](https://pkg.go.dev/github.com/TrebuchetDynamics/goncho/memory#OpenSqlite) | Open a local SQLite store and initialize the service tables used by peer cards, search, context, summaries, and memory tools. |
| [`RunMigrations`](https://pkg.go.dev/github.com/TrebuchetDynamics/goncho#RunMigrations) | Initialize Goncho observation and audit tables on an opened database. |
| [`NewService`](https://pkg.go.dev/github.com/TrebuchetDynamics/goncho#NewService) | Bind a Goncho service to a database, workspace, and observer. |
| [`SetProfile`](https://pkg.go.dev/github.com/TrebuchetDynamics/goncho#Service.SetProfile) | Write stable peer-card facts from the default observer perspective. |
| [`SetProfileForTarget`](https://pkg.go.dev/github.com/TrebuchetDynamics/goncho#Service.SetProfileForTarget) | Write a directional representation: one peer's view of another. |
| [`Profile`](https://pkg.go.dev/github.com/TrebuchetDynamics/goncho#Service.Profile) | Read the current peer card. |
| [`Search`](https://pkg.go.dev/github.com/TrebuchetDynamics/goncho#Service.Search) | Retrieve conclusions or turn evidence relevant to a query. |
| [`Context`](https://pkg.go.dev/github.com/TrebuchetDynamics/goncho#Service.Context) | Assemble an orientation pack for prompt construction. |
| [`Conclude`](https://pkg.go.dev/github.com/TrebuchetDynamics/goncho#Service.Conclude) | Write or delete manual conclusions. |
| [`CreateMessages`](https://pkg.go.dev/github.com/TrebuchetDynamics/goncho#Service.CreateMessages) | Persist session messages as lifecycle evidence. |
| [`OnSessionEnd`](https://pkg.go.dev/github.com/TrebuchetDynamics/goncho#Service.OnSessionEnd) | Consolidate a completed session into summaries. |

## Directional Representation

Memory is not globally true by default. Goncho can distinguish what one observer believes about a target from what another observer believes.

This prevents one perspective from silently becoming universal truth.

:::note[Current setup note]
For a fresh SQLite database, use `memory.OpenSqlite(...)`, then `goncho.RunMigrations(store.DB())`, then pass `store.DB()` to `goncho.NewService`.
:::

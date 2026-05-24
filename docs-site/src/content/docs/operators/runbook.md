---
title: Operator Runbook
description: How to run, verify, back up, and troubleshoot Goncho in an agent host.
---

This runbook is for operators embedding Goncho in a local or server-side agent runtime.

Goncho is intentionally small at the operational layer: one Go library, one SQLite database by default, deterministic migrations, and local verification commands.

Use this runbook with the [Quick Start](/start/quick-start/), [Current Capabilities](/start/current-capabilities/), and [Core API](/reference/core-api/) references when moving from first setup to an operator-owned deployment.

## Operator Contract

A Goncho deployment should make these facts obvious:

| Question | Operator answer |
| --- | --- |
| Which workspace owns this memory? | `Config.WorkspaceID` |
| Which agent perspective writes observations? | `Config.ObserverPeerID` |
| Where is state stored? | SQLite database path or injected `*sql.DB` |
| What tools are exposed to the agent? | Public Goncho tools and/or MCP-style memory tools |
| How is context bounded? | `RecentMessages`, `GetContextMaxTokens`, per-call `MaxTokens` |
| How are risky memories handled? | review queue, stale warnings, quarantine, negative drift anchors |
| How do we prove it works after deploy? | local smoke tests and docs build commands |

## Recommended Runtime Shape

For Gormes-style hosts, prefer the adapter package:

```go
mem, err := gormesgoncho.Open(ctx, gormesgoncho.Config{
    DatabasePath: "/var/lib/gormes/goncho.db",
    WorkspaceID:  "gormes-prod",
    ObserverID:   "gormes",
})
if err != nil {
    return err
}
defer func() { _ = mem.Close(ctx) }()
```

For custom hosts, wire the service directly:

```go
store, err := memory.OpenSqlite("goncho.db", 0, nil)
if err != nil {
    return err
}
if err := goncho.RunMigrations(store.DB()); err != nil {
    return err
}
svc := goncho.NewService(store.DB(), goncho.Config{
    WorkspaceID:    "custom-agent",
    ObserverPeerID: "assistant",
    RecentMessages: 8,
}, nil)
```

Use one database per trust boundary. For example, separate local developer memory from shared team memory unless you have explicit review and ACL controls around the shared runtime.

## Startup Checklist

Run this at process startup or deployment time:

1. Open the SQLite database path expected by the runtime.
2. Run `goncho.RunMigrations(db)` exactly once during initialization.
3. Construct `goncho.NewService` with explicit `WorkspaceID` and `ObserverPeerID`.
4. Register only the tools the host wants the model to call.
5. Emit a startup log line with workspace id, observer id, database path, registered tool names, `Status().Capabilities`, and JSON-friendly operation specs/schemas from `Status().ToolSpecs`.
6. Call `Status().RequireCapabilities(...)` for host-required features such as `context`, `recall`, or `recall_compact`.
7. Run a context smoke call for a known test peer if the environment supports it.

## Tool Exposure Policy

Expose a small surface first.

| Tool | Mutates memory? | Suggested access |
| --- | --- | --- |
| `goncho_context` | No | Safe default. Use before prompt construction. |
| `goncho_search` | No | Safe default. Use for flat explicit memory search. |
| `goncho_recall` | No | Safe default for audit/debug flows that need scored trace, diagnostics, and replay evidence. |
| `goncho_remember` | Yes | Gate behind operator policy or explicit host rules. |
| `goncho_review` | Yes for resolve actions | Operator/system only. |
| `goncho_handoff` | Yes for save actions | Agent or operator, depending on session policy. |
| `store_memory` | Yes | MCP compatibility; expose only if generic memory tools are needed. |
| `retrieve_memory` | No | MCP compatibility. |
| `update_memory` | Yes | Operator/system or reviewed agent action. |
| `forget_memory` | Yes | Operator/system only unless the product has a user-facing deletion flow. |

Prefer `goncho_context`, `goncho_search`, `goncho_recall`, `goncho_remember`, `goncho_review`, and `goncho_handoff` for new Goncho-native hosts. Use MCP-style tools when the host already expects generic memory contracts.

## Health Checks

### Build-time checks

```sh
make release-smoke
```

`make release-smoke` runs release metadata checks, ecosystem smoke, `go test ./...`, `go vet ./...`, `go test -race ./...`, and the docs-site build. For narrower checks, run:

```sh
make release-metadata-smoke
make ecosystem-smoke
go test ./integration/gormes
go test ./...
cd docs-site && npm run build
```

### Local service smoke check

```sh
go test ./... -run TestLocalE2E_ServiceLifecycleBuildsContextFromPublicAPIs
```

### Restart persistence checks

```sh
go test ./... -run TestGonchoPublicToolsRestartE2E
go test ./... -run TestHTTPServiceRestartE2E
```

### Trust checks

```sh
go test ./... -run TestGonchoGoalPromptInjectionImportIsQuarantinedE2E
go test ./... -run TestGonchoGoalStaleCodeClaimRequiresLiveVerificationE2E
go test ./... -run TestGonchoGoalNegativeDriftAnchorWarnsBeforeRepeatedFailureE2E
```

## Backup and Restore

For local SQLite deployments:

1. Stop the agent process or put it into a no-write maintenance mode.
2. Copy the database file and its `-wal`/`-shm` files if WAL is enabled.
3. Store the backup with timestamp, workspace id, and host identity.
4. Restore into a new path first.
5. Run the local smoke test against the restored copy before replacing production state.

Do not edit the SQLite database by hand during an active agent run.

## Import Safety

Imported files can contain prompt-injection-like text. Goncho preserves suspicious imports as skipped evidence and excludes them from trusted context/search.

Operator expectation:

- suspicious text is not deleted silently;
- suspicious text is not promoted as trusted context;
- context responses include unavailable evidence such as `prompt_injection_quarantine`;
- review tooling can inspect unresolved trust issues.

## Staleness and Live Truth

Treat old code memories as claims from a frozen point in time. Before acting on remembered file/function/API claims, use live repository checks or Goncho's verified code context path.

Operator expectation:

- stale code paths should warn as `stale_code_claim`;
- current live paths can still surface;
- risky code memories should not silently steer edits.

## Negative Memory and Drift

Negative memories preserve dead ends and failed paths. Use them to prevent repeated failure, not to block unrelated work.

Operator expectation:

- dead-end memories should be tagged clearly, for example `negative`, `dead-end`, or `drift-anchor`;
- prompts that resemble known failures should produce `negative_drift_anchor` warnings;
- unrelated prompts should not warn.

## Benchmark Operations

Use the LOCOMO benchmark targets when validating retrieval behavior or comparing backends.

Smoke checks for normal local validation:

```sh
make bench-locomo-smoke
make bench-locomo-backends-smoke
```

Full checks for release or research validation:

```sh
make bench-locomo
make bench-locomo-backends
```

Operator rules:

- Treat the Go benchmark harness as the source of truth for scoring.
- Keep external backend adapters isolated; they may only emit retrieved `memory_id` values and scores.
- Keep LOCOMO backend comparison conversation-scoped before scoring so duplicate content in another conversation cannot win by content alone.
- Do not accept LLM judges or answer text as retrieval evidence.
- Do not map external results by content unless the adapter proves the mapping is collision-safe.
- If a backend cannot return stable inserted IDs, leave it `not comparable` and report the exact reason.

Backend probe commands:

```sh
AGENTMEMORY_SOURCE_DIR=/path/to/agentmemory-pr583 python3 scripts/bench_agentmemory_locomo.py --capability
AGENTMEMORY_SOURCE_DIR=/path/to/agentmemory-pr583 python3 scripts/bench_agentmemory_locomo.py --smoke
python3 scripts/bench_mem0_locomo.py --capability
python3 scripts/bench_mem0_locomo.py --smoke
```

Expected current status:

| Backend | Status | Operator meaning |
| --- | --- | --- |
| Goncho | comparable | Local adapter returns stable IDs. |
| BM25 | comparable | Local lexical baseline returns stable IDs. |
| SQLite FTS5 | comparable | Local FTS baseline returns stable IDs. |
| agentmemory | comparable with PR source | Set `AGENTMEMORY_SOURCE_DIR` to PR #583 commit `9b18a80c9d2839b025279978d3f4b5e1f9bc6e74`. Stable IDs work; standalone fallback LOCOMO score is `0.0000`, and this is not the full running server. |
| mem0 | not comparable | Package is not installed locally; no stable-ID run exists. |

Primary outputs:

- `docs/benchmarks/locomo-backend-comparison.md`
- `docs/benchmarks/results/locomo-backend-comparison.json`
- `docs/benchmarks/failures/locomo-backend-comparison.jsonl`
- `docs/benchmarks/external-backend-adapters.md`

## Troubleshooting

| Symptom | Check |
| --- | --- |
| Empty context | Confirm `peer_id`, `session_key`, workspace id, and whether memory was written in this database. |
| Memory appears in wrong runtime | Check `Config.WorkspaceID` and database path. Use separate DBs for separate trust boundaries. |
| Old path appears in context | Run stale code-claim verification and inspect `stale_code_claim` evidence. |
| Imported text steers behavior | Check import status and quarantine evidence. Suspicious imports should be `skipped`. |
| Agent repeats failed fix | Store a negative/dead-end memory and enable drift-anchor checks in the host loop. |
| Review warnings keep appearing | List review items with `goncho_review`; resolve only after operator evidence review. |
| External backend marked not comparable | Confirm it can return the exact inserted `memory_id` in search results. If not, keep it not comparable; do not use content-only matching. |
| Backend comparison scores changed unexpectedly | Re-run `make bench-locomo-backends-smoke`, check `docs/benchmarks/failures/locomo-backend-comparison.jsonl`, and verify the same LOCOMO data files were used. |

## Release Checklist

Before upgrading Goncho in an operator environment:

1. Run `make release-smoke` on the candidate commit.
2. Run the restart persistence checks.
3. Run trust checks for quarantine, stale code claims, and negative drift anchors.
4. Run `make bench-locomo-backends-smoke` if retrieval/backend harness behavior changed.
5. Pin the module version or commit.
6. Back up the current SQLite database.
7. Deploy to a staging agent with a copied database.
8. Confirm startup logs show the expected workspace, observer, DB path, and tools.
9. Promote to production only after the staging agent builds a valid context pack.

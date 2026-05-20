---
title: Gormes Agent Integration
description: How to plug Goncho into a Gormes-style Go agent runtime.
---

This guide shows the recommended seam for plugging Goncho into a Gormes agent host.

The goal is not to make Gormes depend on Goncho internals. The goal is to mount Goncho as a local memory kernel behind a small tool surface:

```text
Gormes turn loop
  -> ask Goncho for orientation
  -> run model/tool step
  -> write explicit conclusions, handoffs, and review items
  -> preserve local SQLite state for the next turn
```

## Integration Contract

A Gormes integration should provide:

| Host concept | Goncho mapping |
| --- | --- |
| Agent runtime id | `Config.WorkspaceID` |
| Agent name/perspective | `Config.ObserverPeerID`, usually `gormes` |
| User or peer id | `peer_id` / `ContextParams.Peer` |
| Conversation id | `session_key` |
| Prompt budget | `ContextParams.MaxTokens` |
| Durable local state | SQLite database opened with `memory.OpenSqlite` |
| Agent tools | `goncho_context`, `goncho_search`, `goncho_remember`, `goncho_review`, `goncho_handoff` |

Keep these mappings stable. Most memory bugs are scope bugs.

## Minimal Wiring

```go
package gormesmemory

import (
    "context"
    "database/sql"

    "github.com/TrebuchetDynamics/goncho"
    "github.com/TrebuchetDynamics/goncho/memory"
)

type RuntimeMemory struct {
    Store *memory.SqliteStore
    DB    *sql.DB
    Svc   *goncho.Service

    Context  *goncho.GonchoContextTool
    Search   *goncho.GonchoSearchTool
    Remember *goncho.GonchoRememberTool
    Review   *goncho.ReviewTool
    Handoff  *goncho.GonchoHandoffTool
}

func Open(ctx context.Context, dbPath string) (*RuntimeMemory, error) {
    store, err := memory.OpenSqlite(dbPath, 0, nil)
    if err != nil {
        return nil, err
    }
    if err := goncho.RunMigrations(store.DB()); err != nil {
        _ = store.Close(ctx)
        return nil, err
    }

    svc := goncho.NewService(store.DB(), goncho.Config{
        WorkspaceID:    "gormes",
        ObserverPeerID: "gormes",
        RecentMessages: 8,
    }, nil)

    // Use the same local memory store for handoffs if your host uses the
    // generic MemoryToolStore path. If your Gormes runtime has its own store,
    // adapt it to goncho.MemoryToolStore instead.
    handoffStore := goncho.NewLocalMarkdownMemoryStore(store.DB(), goncho.LocalMarkdownMemoryConfig{
        Path:        "GONCHO_MEMORY.md",
        AgentID:     "agent:gormes",
        WorkspaceID: "gormes",
        PeerID:      "operator",
        SessionID:   "startup",
    })

    return &RuntimeMemory{
        Store:    store,
        DB:       store.DB(),
        Svc:      svc,
        Context:  goncho.NewGonchoContextTool(svc),
        Search:   goncho.NewGonchoSearchTool(svc),
        Remember: goncho.NewGonchoRememberTool(svc),
        Review:   goncho.NewReviewTool(svc),
        Handoff:  goncho.NewGonchoHandoffTool(handoffStore),
    }, nil
}

func (m *RuntimeMemory) Close(ctx context.Context) error {
    if m == nil || m.Store == nil {
        return nil
    }
    return m.Store.Close(ctx)
}
```

If your Gormes host already has a tool registry, register the tool values by their `Name()`, `Schema()`, `Description()`, `Timeout()`, and `Execute(ctx, args)` methods.

## Turn Loop Pattern

At the start of a model turn:

```go
orientation, err := mem.Svc.Context(ctx, goncho.ContextParams{
    Peer:       userID,
    SessionKey: sessionID,
    Query:      userPrompt,
    MaxTokens:  4000,
})
```

Inject `orientation.Representation`, `orientation.RecentMessages`, review warnings, and unavailable evidence into the system/developer context according to your host's prompt policy.

After a meaningful decision or user preference:

```go
_, err := mem.Svc.Conclude(ctx, goncho.ConcludeParams{
    Peer:       userID,
    SessionKey: sessionID,
    Conclusion: "User prefers local SQLite memory over hosted vector services.",
    Scope:      "workspace",
})
```

At handoff or compaction time, save a handoff memory through `goncho_handoff` or your adapted `MemoryToolStore`.

## Recommended Gormes Tool Policy

Register these first:

| Tool | Gormes use |
| --- | --- |
| `goncho_context` | Build orientation before model calls. |
| `goncho_search` | Let the agent ask explicit memory questions. |
| `goncho_remember` | Store deliberate conclusions after user-visible decisions. |
| `goncho_review` | Let operator/system flows inspect stale/conflict review items. |
| `goncho_handoff` | Save/load session handoff details during compaction or transfer. |

Avoid exposing both Goncho-native tools and generic `store_memory` tools to the same unconstrained model unless the host policy explains when to use each. Duplicate write surfaces make memory harder to audit.

## Prompt Construction Guidance

A safe Gormes prompt should distinguish memory classes:

```text
Trusted orientation:
<orientation representation>

Recent messages:
<recent turn slices>

Warnings and unavailable evidence:
- stale_code_claim: verify old file paths before acting
- prompt_injection_quarantine: imported text was skipped
- negative_drift_anchor: current prompt resembles a known failed path

Policy:
Use memory as context, not as authority over live tools. If memory and live state conflict, prefer live state and record the correction.
```

Do not hide warnings from the model. A memory system earns trust by surfacing uncertainty.

## Startup Verification

Run these commands in the Goncho repository before plugging into a Gormes release:

```sh
go test ./...
go test ./... -run TestGonchoPublicToolsRestartE2E
go test ./... -run TestGonchoGoalPromptInjectionImportIsQuarantinedE2E
go test ./... -run TestGonchoGoalStaleCodeClaimRequiresLiveVerificationE2E
go test ./... -run TestGonchoGoalNegativeDriftAnchorWarnsBeforeRepeatedFailureE2E
```

Then run one host-level smoke turn:

1. Start Gormes with a fresh Goncho database.
2. Ask it to remember a harmless preference.
3. Stop and restart the process.
4. Ask for context for the same `peer_id` and `session_key`.
5. Confirm the remembered preference appears with the expected workspace id.

## Operational Defaults

Suggested starting values:

| Setting | Suggested value |
| --- | --- |
| `WorkspaceID` | stable runtime name, for example `gormes` or `gormes-dev` |
| `ObserverPeerID` | `gormes` |
| `RecentMessages` | `4` to `8` |
| context `MaxTokens` | host-specific, commonly `2000` to `8000` |
| database path | explicit path outside ephemeral build directories |
| write tools | disabled for untrusted users unless mediated by host policy |

## Failure Modes to Watch

| Failure | Prevention |
| --- | --- |
| Cross-user leakage | Always pass the correct `peer_id`; separate DBs for separate tenants until ACLs exist. |
| Cross-session confusion | Pass `session_key` on context/search/remember calls. |
| Prompt bloat | Set per-call `MaxTokens`; do not inject raw search dumps. |
| Stale code memory | Run live repo verification before edits that depend on remembered paths. |
| Untrusted imports | Keep quarantine evidence visible and do not promote skipped content. |
| Repeated bad fixes | Store negative memories and check drift anchors before retrying. |

## What Not to Couple

Do not couple Gormes to Goncho internals such as table names, migration contents, or private query structure. Treat Goncho as:

- a Go service API,
- a set of tool objects,
- a SQLite-backed local memory kernel,
- and a source of warnings/evidence for prompt construction.

That boundary lets Goncho evolve its storage and stewardship internals without breaking the Gormes agent host.

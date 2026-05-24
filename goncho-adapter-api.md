# Goncho Adapter API for gormes-agent

## Purpose

`gormes-agent` should use Goncho as an embedded Go memory service through a thin adapter layer.

The adapter owns host-specific concerns:

- profile/workspace paths,
- startup and migrations,
- tool registration,
- runtime hook forwarding,
- context/recall injection,
- host-owned side effects such as git, process control, network calls, and UI.

Goncho owns memory behavior:

- durable profile facts,
- conclusions,
- observations,
- recall/search/context,
- slots,
- local consolidation,
- resources/prompts,
- action graph metadata,
- deterministic snapshot metadata,
- image refs/checksums.

## Import shape

```go
import (
    goncho "github.com/TrebuchetDynamics/goncho/service"
    "github.com/TrebuchetDynamics/goncho/memory"
)
```

## Startup contract

At gormes-agent startup:

```go
store, err := memory.OpenSqlite(dbPath, 0, nil)
if err != nil { return err }

if err := goncho.RunMigrations(store.DB()); err != nil { return err }

svc := goncho.NewService(store.DB(), goncho.Config{
    WorkspaceID:    workspaceID,
    ObserverPeerID: "gormes",
    RecentMessages: 4,
    VectorStore:    optionalVectorStore, // nil is valid
}, logger)
```

Recommended paths:

```text
.gormes/profiles/<profile_id>/goncho.db
.gormes/profiles/<profile_id>/GONCHO_MEMORY.md
```

Do not share one private profile DB across unrelated profiles unless the adapter intentionally uses workspace/shared scope.

## Runtime flow

Recommended per-turn flow:

```text
user prompt
  -> adapter calls CaptureHostHook(UserPrompt)
  -> adapter calls Context or Recall
  -> gormes-agent builds model prompt from returned evidence
  -> model/tool loop runs
  -> adapter forwards tool results/failures with CaptureHostHook(PostToolUse/Failure)
  -> adapter forwards assistant response
  -> on session end, adapter forwards SessionEnd and may run consolidation
```

## Hook forwarding

Map gormes-agent runtime events to Goncho host-neutral hooks.

```go
_, err := svc.CaptureHostHook(ctx, goncho.HostHookEvent{
    Event:      goncho.HostHookUserPrompt,
    Host:       "gormes-agent",
    ProfileID:  profileID,
    PeerID:     peerID,
    SessionKey: sessionID,
    Content:    prompt,
})
```

Suggested mapping:

| gormes-agent event | Goncho event |
| --- | --- |
| user prompt received | `HostHookUserPrompt` |
| assistant response completed | `HostHookAssistantResponse` |
| tool call returned success | `HostHookPostToolUse` with `Success: true` |
| tool call failed | `HostHookPostToolUse` with `Success: false`, or `HostHookFailure` |
| compaction/handoff | `HostHookCompact` |
| session closed | `HostHookSessionEnd` |

Tool result example:

```go
ok := false
_, err := svc.CaptureHostHook(ctx, goncho.HostHookEvent{
    Event:      goncho.HostHookPostToolUse,
    Host:       "gormes-agent",
    ProfileID:  profileID,
    PeerID:     peerID,
    SessionKey: sessionID,
    ToolName:   "bash",
    Input:      "go test ./...",
    Output:     "test failed",
    Success:    &ok,
})
```

## Context and recall

Use `Context` for compact orientation packs.

```go
ctxResult, err := svc.Context(ctx, goncho.ContextParams{
    ProfileID:  profileID,
    Peer:       peerID,
    SessionKey: sessionID,
    Query:      userPrompt,
    MaxTokens:  4000,
})
```

Use `Recall` when the agent needs auditable provenance.

```go
trace, err := svc.Recall(ctx, goncho.RecallQuery{
    WorkspaceID: workspaceID,
    Peer:        peerID,
    Query:       userPrompt,
    SessionKey:  sessionID,
    ScopeID:     goncho.MemoryScopeProfile,
    Limit:       5,
    MaxTokens:   2000,
})
```

Adapter prompt rule:

- include selected memory content,
- include `memory_id`/provenance when available,
- include warnings/degraded evidence,
- instruct the model to verify live state before acting.

## Tool registration

Register Goncho public tools into gormes-agent's tool registry.

```go
tools := []Tool{
    goncho.NewGonchoContextTool(svc),
    goncho.NewGonchoSearchTool(svc),
    goncho.NewGonchoRecallTool(svc),
    goncho.NewGonchoRememberTool(svc),
}
```

Expected tool names:

- `goncho_context`
- `goncho_search`
- `goncho_recall`
- `goncho_remember`

Optional tools/resources:

- review tool,
- handoff tool,
- `NewMemoryResourceRegistry` for resource/prompt discovery.

## Resources and prompts

Use the Go-neutral registry when gormes-agent wants status/profile/latest/graph/recall prompt data without MCP.

```go
registry := goncho.NewMemoryResourceRegistry(svc)
resources := registry.Descriptors()
status, err := registry.Read(ctx, goncho.MemoryResourceRequest{
    URI:  "goncho://status",
    Peer: peerID,
})
```

Supported URIs:

- `goncho://status`
- `goncho://profile`
- `goncho://latest`
- `goncho://graph/stats`
- `goncho://recall/prompt`

## Durable writes

### Explicit conclusions

Use for durable facts/claims extracted by the host or explicitly requested by the user.

```go
_, err := svc.Conclude(ctx, goncho.ConcludeParams{
    ProfileID:  profileID,
    Peer:       peerID,
    SessionKey: sessionID,
    Scope:      goncho.MemoryScopeProfile,
    Conclusion: "User prefers concise answers.",
})
```

### Slots

Use slots for named durable preferences/facts.

```go
slot, err := svc.CreateMemorySlot(ctx, goncho.MemorySlotParams{
    ProfileID: profileID,
    Peer:      peerID,
    Scope:     goncho.MemoryScopeProfile,
    Name:      "reply_style",
    Kind:      "preference",
    Value:     "concise, no filler",
})
```

Slot API:

- `CreateMemorySlot`
- `GetMemorySlot`
- `ListMemorySlots`
- `AppendMemorySlot`
- `ReplaceMemorySlot`
- `DeleteMemorySlot`

Slots are revisioned, tombstoned on delete, and audited through observations.

## Consolidation

Run explicit local consolidation at safe session boundaries, not continuously in the hot path.

```go
result, err := svc.ExecuteFourTierConsolidation(ctx, goncho.FourTierConsolidationParams{
    ProfileID:  profileID,
    Peer:       peerID,
    SessionKey: sessionID,
    Scope:      goncho.MemoryScopeProfile,
})
```

This writes four tiers:

- `working`,
- `episodic`,
- `semantic`,
- `procedural`.

Each item carries consolidation provenance.

## Action graph

Use local action graph for agent coordination metadata.

```go
_, err := svc.UpsertAction(ctx, goncho.ActionParams{
    ProfileID: profileID,
    Peer:      peerID,
    ActionID:  "write-tests",
    Title:     "Write Goncho adapter tests",
    DependsOn: []string{"design-adapter"},
})

graph, err := svc.ReadActionGraph(ctx, goncho.ActionGraphQuery{
    ProfileID: profileID,
    Peer:      peerID,
})
```

Use:

- `graph.Frontier` for unblocked actions,
- `graph.NextAction` for the next local recommendation,
- `SignalAction` for blocked/ready/needs-review signals.

Server leases are intentionally not part of this local slice.

## Snapshots

Goncho only builds deterministic metadata. gormes-agent owns git operations.

```go
manifest, err := svc.ExportSnapshotManifest(ctx, goncho.SnapshotParams{
    ProfileID: profileID,
    Peer:      peerID,
})
```

Use:

- `ExportSnapshotManifest`,
- `DiffSnapshotManifests`,
- `BuildSnapshotRollbackMetadata`.

Do not expect Goncho to run `git add`, `git commit`, `git diff`, or checkout/rollback.

## Image refs

Store image references and checksums now; real embeddings can be added later.

```go
img, err := svc.StoreImageMemory(ctx, goncho.ImageMemoryParams{
    ProfileID:  profileID,
    Peer:       peerID,
    SessionKey: sessionID,
    ImageRef:   "file://screenshots/login-error.png",
    Checksum:   "sha256:...",
    AltText:    "Login page showing invalid token error",
    Metadata:   map[string]string{"media_type": "image/png"},
})
```

Search by checksum/ref/alt text:

```go
imgs, err := svc.SearchImageMemories(ctx, goncho.ImageMemoryQuery{
    ProfileID: profileID,
    Peer:      peerID,
    Query:     "invalid token",
})
```

`EmbeddingStatus` is `deferred` until a future image embedding adapter exists.

## Optional vector store

If gormes-agent has a local embedding index, implement `goncho.VectorStore`.

```go
type LocalVectorStore struct{}

func (v LocalVectorStore) Search(ctx context.Context, q goncho.VectorSearchQuery) ([]goncho.VectorSearchHit, error) {
    // Query local embedding index; no network required.
    return hits, nil
}
```

Then pass it into config:

```go
svc := goncho.NewService(db, goncho.Config{
    WorkspaceID:    workspaceID,
    ObserverPeerID: "gormes",
    VectorStore:    LocalVectorStore{},
}, logger)
```

Goncho fuses vector hits into Recall as `semantic` provenance through RRF.

## Isolation rules

The adapter must always pass:

- `WorkspaceID`,
- `ProfileID`,
- `Peer`,
- `SessionKey`.

Default recommendation:

```text
profile_id present + no explicit shared scope = private profile memory
```

Use explicit scopes when widening:

- `profile` for private profile memory,
- `workspace` for workspace-shared memory,
- `shared` for intentional shared memory,
- `global` only for truly global facts.

## Do not do this

- Do not shell out to Goncho CLI for normal runtime memory.
- Do not write Goncho SQLite tables directly.
- Do not treat memory as permission to skip live verification.
- Do not mix profiles accidentally.
- Do not let Goncho own git operations.
- Do not expose private profile memory in shared prompts without scope evidence.

## Minimal adapter interface

Suggested gormes-agent adapter surface:

```go
type GonchoAdapter interface {
    Start(ctx context.Context) error
    Close(ctx context.Context) error

    CapturePrompt(ctx context.Context, profileID, peerID, sessionID, prompt string) error
    CaptureAssistant(ctx context.Context, profileID, peerID, sessionID, response string) error
    CaptureTool(ctx context.Context, profileID, peerID, sessionID, tool, input, output string, success bool) error
    CaptureSessionEnd(ctx context.Context, profileID, peerID, sessionID, summary string) error

    Context(ctx context.Context, profileID, peerID, sessionID, query string, maxTokens int) (goncho.ContextResult, error)
    Recall(ctx context.Context, profileID, peerID, sessionID, query string, limit int) (goncho.RecallTrace, error)

    Remember(ctx context.Context, profileID, peerID, sessionID, content string) error
    GetSlot(ctx context.Context, profileID, peerID, name string) (goncho.MemorySlot, error)
}
```

## Validation checklist

Before shipping the adapter:

- startup runs `RunMigrations`,
- profile-local DB path is correct,
- hooks are forwarded for prompt/tool/failure/session end,
- `Context` or `Recall` is called before memory-dependent answers,
- tool registration exposes Goncho tools,
- profile isolation tests pass,
- `go test ./...` passes,
- no direct SQLite table writes from gormes-agent.

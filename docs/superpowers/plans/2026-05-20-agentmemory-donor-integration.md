# Agentmemory Donor Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the best agentmemory-inspired integration surfaces to Goncho while preserving Goncho's stricter trust model: evidence before belief, live verification before action, and a small public tool surface.

**Architecture:** Implement this as eight independent vertical slices, each proven by failing tests first. Agentmemory is the integration/interface donor; Goncho remains the source of truth for local-first SQLite storage, auditability, scoped temporal beliefs, quarantine, review, and verification warnings.

**Tech Stack:** Go, SQLite, existing Goncho `Service`, existing public tools in `goncho_public_tools.go`, existing observation/audit/review/storage patterns, deterministic `go test ./...` verification.

---

## Source Evidence

Reference tree inspected:

- `docs/opensource-memory-systems/agentmemory/README.md`
- `docs/opensource-memory-systems/agentmemory/src/types.ts`
- `docs/opensource-memory-systems/agentmemory/src/index.ts`
- `docs/opensource-memory-systems/agentmemory/src/mcp/tools-registry.ts`
- `docs/opensource-memory-systems/METAANALYSIS-MEMORY-SYSTEMS.md`

Relevant Goncho files inspected:

- `observations.go`
- `service.go`
- `types.go`
- `goncho_public_tools.go`
- `memory_tools.go`
- `review.go`
- `review_tool.go`
- `audit.go`
- `code_claim_verification.go`

Current Goncho capabilities already present:

- `Service.Observe` and `ListObservations` for evidence capture.
- Redaction flags and counts on `Observation`.
- Audit records for observations.
- `Search`, `Context`, `Conclude`, profile, review, handoff, local memory tools.
- Prompt-injection quarantine and live code-claim verification in existing code paths.

Design rule for all tasks:

```text
agentmemory pattern -> Goncho evidence/claim/belief equivalent -> failing test -> minimal implementation -> go test ./... -> atomic commit
```

---

## Public Surface Boundary

Do not copy agentmemory's 53-tool MCP surface as Goncho defaults.

Keep Goncho core tools small:

- `goncho_context`
- `goncho_search`
- `goncho_remember`
- `goncho_review`
- `goncho_handoff`
- `goncho_file_history` introduced in Task 3
- `goncho_verify` introduced in Task 7

Optional extended tools, only after core APIs stabilize:

- `goncho_timeline`
- `goncho_relations`
- `goncho_profile`
- `goncho_export`
- `goncho_snapshot`
- `goncho_hooks_observe`

---

## Planned File Structure

Task 1: Hook ingestion adapter

- Create: `hook_ingestion.go`
- Create: `hook_ingestion_test.go`
- Modify: `observations.go` only if a missing `ObservationKind` is required.

Task 2: Privacy/redaction gate

- Create: `privacy_filter.go`
- Create: `privacy_filter_test.go`
- Modify: `observations.go` only to call the shared filter from normalization if current redaction is duplicated.

Task 3: File history API and tool

- Create: `file_history.go`
- Create: `file_history_test.go`
- Modify: `goncho_public_tools.go`
- Modify: `metaanalysis_public_tools_test.go` or create `file_history_tool_test.go`.

Task 4: Progressive Search plus Expand

- Create: `search_expand.go`
- Create: `search_expand_test.go`
- Modify: `types.go`
- Modify: `service.go`
- Modify: `goncho_public_tools.go` only if `goncho_search` needs `mode` and `expand_ids` fields.

Task 5: Project profile/context boot pack

- Create: `project_profile.go`
- Create: `project_profile_test.go`
- Modify: `types.go`
- Modify: `service.go` or keep as methods in `project_profile.go`.

Task 6: Relations/query graph

- Create: `relations.go`
- Create: `relations_test.go`
- Create migration file if this repo's migration pattern requires one.
- Modify: `types.go` if shared relation result types belong there.

Task 7: Verify/provenance API and tool

- Create: `verify.go`
- Create: `verify_test.go`
- Modify: `goncho_public_tools.go` to add `GonchoVerifyTool`.
- Modify: `metaanalysis_public_tools_test.go` or create `verify_tool_test.go`.

Task 8: Export/snapshot

- Create: `export_snapshot.go`
- Create: `export_snapshot_test.go`
- Modify migrations only if snapshot metadata needs a table.

Documentation status updates after each public surface change:

- Modify: `README.md`
- Modify: `docs/opensource-memory-systems/METAANALYSIS-MEMORY-SYSTEMS.md` only if the feature matrix/status narrative needs updating.

---

## Task 1: Hook Ingestion Adapter

**Capability copied from agentmemory:** Hook-native capture via `SessionStart`, `UserPromptSubmit`, `PreToolUse`, `PostToolUse`, `PostToolUseFailure`, `PreCompact`, `SubagentStart`, `SubagentStop`, `Stop`, and `SessionEnd`.

**Goncho interpretation:** Hooks become local evidence through `Service.Observe`; they do not become trusted memory until later extraction, review, or explicit conclusion.

**Files:**

- Create: `hook_ingestion.go`
- Create: `hook_ingestion_test.go`
- Existing dependency: `observations.go`

- [ ] **Step 1: Write failing adapter test**

Add `hook_ingestion_test.go` with a test that proves an agentmemory-style hook maps to Goncho observation evidence.

```go
package goncho

import (
    "context"
    "testing"
    "time"

    "github.com/TrebuchetDynamics/goncho/memory"
)

func TestHookIngestionStoresPostToolUseAsEvidence(t *testing.T) {
    ctx := context.Background()
    store, err := memory.OpenSqlite(t.TempDir()+"/memory.db", 0, nil)
    if err != nil {
        t.Fatal(err)
    }
    defer store.Close(ctx)
    if err := RunMigrations(store.DB()); err != nil {
        t.Fatal(err)
    }
    svc := NewService(store.DB(), Config{WorkspaceID: "ws", ObserverPeerID: "agent"}, nil)

    observedAt := time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)
    result, err := svc.ObserveHook(ctx, HookObservationParams{
        Hook:       HookPostToolUse,
        SessionKey: "session-1",
        PeerID:     "agent",
        ToolName:   "read",
        ToolInput:  `{"path":"README.md"}`,
        ToolOutput: "# Goncho",
        ObservedAt: observedAt,
    })
    if err != nil {
        t.Fatal(err)
    }
    if result.Observation.Kind != ObservationKindToolResult {
        t.Fatalf("kind = %q, want %q", result.Observation.Kind, ObservationKindToolResult)
    }
    if result.Observation.Metadata["hook"] != string(HookPostToolUse) {
        t.Fatalf("hook metadata = %q", result.Observation.Metadata["hook"])
    }
    if result.Observation.Metadata["tool_name"] != "read" {
        t.Fatalf("tool metadata = %q", result.Observation.Metadata["tool_name"])
    }
    if result.Observation.SessionKey != "session-1" {
        t.Fatalf("session = %q", result.Observation.SessionKey)
    }
}
```

- [ ] **Step 2: Run narrow test and confirm RED**

Run:

```bash
go test ./... -run TestHookIngestionStoresPostToolUseAsEvidence
```

Expected before implementation:

```text
undefined: HookObservationParams
undefined: HookPostToolUse
svc.ObserveHook undefined
```

- [ ] **Step 3: Implement minimal adapter**

Create `hook_ingestion.go` with:

```go
package goncho

import (
    "context"
    "strings"
    "time"
)

type HookType string

const (
    HookSessionStart       HookType = "session_start"
    HookPromptSubmit       HookType = "prompt_submit"
    HookPreToolUse         HookType = "pre_tool_use"
    HookPostToolUse        HookType = "post_tool_use"
    HookPostToolUseFailure HookType = "post_tool_failure"
    HookPreCompact         HookType = "pre_compact"
    HookSubagentStart      HookType = "subagent_start"
    HookSubagentStop       HookType = "subagent_stop"
    HookStop               HookType = "stop"
    HookSessionEnd         HookType = "session_end"
)

type HookObservationParams struct {
    ID          string
    Hook        HookType
    WorkspaceID string
    PeerID      string
    SessionKey  string
    ContextID   string
    ToolName    string
    ToolInput   string
    ToolOutput  string
    UserPrompt  string
    Assistant   string
    Error       string
    Metadata    map[string]string
    ObservedAt  time.Time
}

func (s *Service) ObserveHook(ctx context.Context, p HookObservationParams) (ObservationResult, error) {
    metadata := map[string]string{"hook": string(p.Hook)}
    for k, v := range p.Metadata {
        metadata[k] = v
    }
    if strings.TrimSpace(p.ToolName) != "" {
        metadata["tool_name"] = strings.TrimSpace(p.ToolName)
    }
    kind := observationKindForHook(p.Hook, p.Error)
    return s.Observe(ctx, ObservationParams{
        ID:          p.ID,
        Kind:        kind,
        WorkspaceID: firstPublicNonEmpty(p.WorkspaceID, s.workspaceID),
        PeerID:      p.PeerID,
        SessionKey:  p.SessionKey,
        ContextID:   p.ContextID,
        Input:       firstPublicNonEmpty(p.UserPrompt, p.ToolInput),
        Output:      firstPublicNonEmpty(p.Error, p.Assistant, p.ToolOutput),
        Metadata:    metadata,
        ObservedAt:  p.ObservedAt,
        Reason:      "hook ingestion",
    })
}

func observationKindForHook(h HookType, errText string) ObservationKind {
    if strings.TrimSpace(errText) != "" {
        return ObservationKindToolError
    }
    switch h {
    case HookSessionStart:
        return ObservationKindSessionStart
    case HookPromptSubmit:
        return ObservationKindUserPrompt
    case HookPreToolUse:
        return ObservationKindToolCall
    case HookPostToolUse:
        return ObservationKindToolResult
    case HookPostToolUseFailure:
        return ObservationKindToolError
    case HookPreCompact:
        return ObservationKindCompact
    case HookSessionEnd, HookStop:
        return ObservationKindSessionEnd
    default:
        return ObservationKindCustom
    }
}
```

- [ ] **Step 4: Run narrow test and confirm GREEN**

Run:

```bash
go test ./... -run TestHookIngestionStoresPostToolUseAsEvidence
```

Expected:

```text
ok  github.com/TrebuchetDynamics/goncho
```

- [ ] **Step 5: Run full verification**

Run:

```bash
go test ./...
```

Expected: all packages pass. If unrelated WIP fails, document exact first failing package and stderr line before reporting partial completion.

- [ ] **Step 6: Commit slice**

```bash
git add hook_ingestion.go hook_ingestion_test.go
git commit -m "feat: add hook ingestion evidence adapter"
```

---

## Task 2: Privacy and Redaction Gate

**Capability copied from agentmemory:** Privacy filter before persistence: strip API keys, secrets, and private blocks before indexing or storage.

**Goncho interpretation:** Redaction must happen before durable evidence is written; dangerous content may be preserved only as redacted/quarantined evidence with audit metadata.

**Files:**

- Create: `privacy_filter.go`
- Create: `privacy_filter_test.go`
- Modify: `observations.go` only if its existing redaction should call the shared filter.

- [ ] **Step 1: Write failing privacy filter test**

Add `privacy_filter_test.go`:

```go
package goncho

import "testing"

func TestPrivacyFilterRedactsSecretsAndPrivateBlocks(t *testing.T) {
    input := "token sk-1234567890abcdef and <private>wallet seed words</private>"
    out := FilterPrivateText(input)
    if out.Text == input {
        t.Fatalf("expected redaction")
    }
    if out.RedactionCount != 2 {
        t.Fatalf("redaction count = %d, want 2", out.RedactionCount)
    }
    if containsAny(out.Text, []string{"sk-1234567890abcdef", "wallet seed words"}) {
        t.Fatalf("redacted text leaked secret: %q", out.Text)
    }
}

func containsAny(s string, needles []string) bool {
    for _, n := range needles {
        if strings.Contains(s, n) {
            return true
        }
    }
    return false
}
```

Import `strings` in the test file.

- [ ] **Step 2: Run narrow test and confirm RED**

Run:

```bash
go test ./... -run TestPrivacyFilterRedactsSecretsAndPrivateBlocks
```

Expected before implementation:

```text
undefined: FilterPrivateText
```

- [ ] **Step 3: Implement minimal shared filter**

Create `privacy_filter.go`:

```go
package goncho

import "regexp"

type PrivacyFilterResult struct {
    Text           string
    Redacted       bool
    RedactionCount int
}

var privateBlockPattern = regexp.MustCompile(`(?is)<private>.*?</private>`)
var apiKeyPattern = regexp.MustCompile(`\b(sk-[A-Za-z0-9_-]{12,}|ghp_[A-Za-z0-9_]{12,}|xox[baprs]-[A-Za-z0-9-]{12,})\b`)

func FilterPrivateText(input string) PrivacyFilterResult {
    text := input
    count := 0
    text = privateBlockPattern.ReplaceAllStringFunc(text, func(string) string {
        count++
        return "[REDACTED_PRIVATE]"
    })
    text = apiKeyPattern.ReplaceAllStringFunc(text, func(string) string {
        count++
        return "[REDACTED_SECRET]"
    })
    return PrivacyFilterResult{Text: text, Redacted: count > 0, RedactionCount: count}
}
```

- [ ] **Step 4: Run narrow test and confirm GREEN**

Run:

```bash
go test ./... -run TestPrivacyFilterRedactsSecretsAndPrivateBlocks
```

Expected: pass.

- [ ] **Step 5: Integrate filter into observation normalization with a failing persistence test**

Add a second test in `privacy_filter_test.go` that calls `svc.Observe` with secret-bearing input and confirms `Observation.Redacted == true`, `Observation.RedactionCount > 0`, and stored `Input` does not contain the secret.

- [ ] **Step 6: Run persistence test and confirm RED if existing normalization does not use the shared filter**

Run:

```bash
go test ./... -run TestObservationUsesPrivacyFilterBeforePersistence
```

Expected if not yet wired: fail because `Redacted` is false or secret remains.

- [ ] **Step 7: Wire observation normalization to shared filter**

In `observations.go`, use `FilterPrivateText` for input and output normalization. Preserve existing truncation, checksum, and audit behavior. Combine existing redaction counters with the shared result if current code already redacts other patterns.

- [ ] **Step 8: Full verification and commit**

```bash
go test ./...
git add privacy_filter.go privacy_filter_test.go observations.go
git commit -m "feat: centralize privacy redaction before memory persistence"
```

---

## Task 3: File History API and Public Tool

**Capability copied from agentmemory:** `memory_file_history` for past observations about specific files.

**Goncho interpretation:** File history returns evidence and live verification warnings; it should not claim old observations are current truth.

**Files:**

- Create: `file_history.go`
- Create: `file_history_test.go`
- Modify: `goncho_public_tools.go`
- Create: `file_history_tool_test.go`

- [ ] **Step 1: Write failing service test**

Add `file_history_test.go`:

```go
package goncho

import (
    "context"
    "testing"

    "github.com/TrebuchetDynamics/goncho/memory"
)

func TestFileHistoryReturnsObservationsMentioningFile(t *testing.T) {
    ctx := context.Background()
    store, err := memory.OpenSqlite(t.TempDir()+"/memory.db", 0, nil)
    if err != nil { t.Fatal(err) }
    defer store.Close(ctx)
    if err := RunMigrations(store.DB()); err != nil { t.Fatal(err) }
    svc := NewService(store.DB(), Config{WorkspaceID: "ws", ObserverPeerID: "agent"}, nil)

    _, err = svc.Observe(ctx, ObservationParams{
        Kind: ObservationKindToolResult,
        SessionKey: "s1",
        Input: `{"path":"README.md"}`,
        Output: "read README.md and found install docs",
        Metadata: map[string]string{"file": "README.md"},
    })
    if err != nil { t.Fatal(err) }

    got, err := svc.FileHistory(ctx, FileHistoryParams{Files: []string{"README.md"}, Limit: 10})
    if err != nil { t.Fatal(err) }
    if got.Count != 1 {
        t.Fatalf("count = %d, want 1", got.Count)
    }
    if got.Items[0].File != "README.md" {
        t.Fatalf("file = %q", got.Items[0].File)
    }
}
```

- [ ] **Step 2: Run narrow test and confirm RED**

```bash
go test ./... -run TestFileHistoryReturnsObservationsMentioningFile
```

Expected:

```text
svc.FileHistory undefined
undefined: FileHistoryParams
```

- [ ] **Step 3: Implement minimal service API**

Create `file_history.go`:

```go
package goncho

import (
    "context"
    "strings"
)

type FileHistoryParams struct {
    Files            []string
    SessionKey       string
    ExcludeSessionID string
    Limit            int
}

type FileHistoryItem struct {
    File        string      `json:"file"`
    Observation Observation `json:"observation"`
    Warning     string      `json:"warning,omitempty"`
}

type FileHistoryResult struct {
    Items []FileHistoryItem `json:"items"`
    Count int               `json:"count"`
}

func (s *Service) FileHistory(ctx context.Context, p FileHistoryParams) (FileHistoryResult, error) {
    limit := p.Limit
    if limit <= 0 { limit = 20 }
    observations, err := s.ListObservations(ctx, ObservationQuery{WorkspaceID: s.workspaceID, SessionKey: p.SessionKey, Limit: 500})
    if err != nil { return FileHistoryResult{}, err }
    wanted := make([]string, 0, len(p.Files))
    for _, f := range p.Files {
        f = strings.TrimSpace(f)
        if f != "" { wanted = append(wanted, f) }
    }
    out := FileHistoryResult{}
    for _, obs := range observations.Observations {
        if p.ExcludeSessionID != "" && obs.SessionKey == p.ExcludeSessionID { continue }
        haystack := obs.Input + "\n" + obs.Output + "\n" + obs.Metadata["file"]
        for _, f := range wanted {
            if strings.Contains(haystack, f) {
                out.Items = append(out.Items, FileHistoryItem{File: f, Observation: obs})
                break
            }
        }
        if len(out.Items) >= limit { break }
    }
    out.Count = len(out.Items)
    return out, nil
}
```

- [ ] **Step 4: Add public tool test**

Add `file_history_tool_test.go` to instantiate `NewGonchoFileHistoryTool(svc)` and execute JSON:

```json
{"files":"README.md","limit":10}
```

Assert returned JSON has `success:true`, `action:"file_history"`, and `count:1`.

- [ ] **Step 5: Implement public tool**

Modify `goncho_public_tools.go` to add:

- `type GonchoFileHistoryTool struct{ svc *Service }`
- `NewGonchoFileHistoryTool`
- `Name() string { return "goncho_file_history" }`
- schema fields: `files`, `session_key`, `exclude_session_id`, `limit`
- `Execute` calling `svc.FileHistory`.

- [ ] **Step 6: Full verification and commit**

```bash
go test ./...
git add file_history.go file_history_test.go file_history_tool_test.go goncho_public_tools.go
git commit -m "feat: add file history memory tool"
```

---

## Task 4: Progressive Search plus Expand

**Capability copied from agentmemory:** `memory_smart_search` supports compact search results and expansion by ID.

**Goncho interpretation:** Search should support compact mode by default for agents, with explicit expansion for evidence/provenance details.

**Files:**

- Create: `search_expand.go`
- Create: `search_expand_test.go`
- Modify: `types.go`
- Modify: `service.go`
- Modify: `goncho_public_tools.go` only if tool JSON needs `mode` and `expand_ids`.

- [ ] **Step 1: Write failing compact search test**

Add `search_expand_test.go`:

```go
package goncho

import (
    "context"
    "testing"

    "github.com/TrebuchetDynamics/goncho/memory"
)

func TestSearchCompactModeReturnsExpandableIDs(t *testing.T) {
    ctx := context.Background()
    store, err := memory.OpenSqlite(t.TempDir()+"/memory.db", 0, nil)
    if err != nil { t.Fatal(err) }
    defer store.Close(ctx)
    if err := RunMigrations(store.DB()); err != nil { t.Fatal(err) }
    svc := NewService(store.DB(), Config{WorkspaceID: "ws", ObserverPeerID: "agent"}, nil)
    if _, err := svc.Conclude(ctx, ConcludeParams{Peer: "agent", Conclusion: "Use SQLite for local memory tests"}); err != nil { t.Fatal(err) }

    got, err := svc.Search(ctx, SearchParams{Peer: "agent", Query: "SQLite", Limit: 5, Mode: SearchModeCompact})
    if err != nil { t.Fatal(err) }
    if len(got.Results) == 0 { t.Fatal("expected compact search result") }
    if got.Results[0].ExpandableID == "" { t.Fatalf("missing expandable id: %+v", got.Results[0]) }
}
```

- [ ] **Step 2: Run narrow test and confirm RED**

```bash
go test ./... -run TestSearchCompactModeReturnsExpandableIDs
```

Expected:

```text
unknown field Mode in struct literal of type SearchParams
undefined: SearchModeCompact
got.Results[0].ExpandableID undefined
```

- [ ] **Step 3: Extend types minimally**

Modify `types.go`:

```go
type SearchMode string

const (
    SearchModeFull    SearchMode = "full"
    SearchModeCompact SearchMode = "compact"
)
```

Add fields:

```go
Mode SearchMode `json:"mode,omitempty"`
```

to `SearchParams`, and:

```go
ExpandableID string `json:"expandable_id,omitempty"`
```

to `SearchHit`.

- [ ] **Step 4: Populate `ExpandableID` in `Service.Search`**

In `service.go`, after constructing each `SearchHit`, set `ExpandableID` to a stable string such as `conclusion:<id>` when `ID` is present or `session:<session_key>` for session-derived hits.

- [ ] **Step 5: Write failing expand test**

Add test:

```go
func TestExpandSearchHitReturnsFullEvidence(t *testing.T) {
    ctx := context.Background()
    store, err := memory.OpenSqlite(t.TempDir()+"/memory.db", 0, nil)
    if err != nil { t.Fatal(err) }
    defer store.Close(ctx)
    if err := RunMigrations(store.DB()); err != nil { t.Fatal(err) }
    svc := NewService(store.DB(), Config{WorkspaceID: "ws", ObserverPeerID: "agent"}, nil)
    if _, err := svc.Conclude(ctx, ConcludeParams{Peer: "agent", Conclusion: "Progressive search expands evidence"}); err != nil { t.Fatal(err) }
    search, err := svc.Search(ctx, SearchParams{Peer: "agent", Query: "expands", Limit: 1, Mode: SearchModeCompact})
    if err != nil { t.Fatal(err) }
    expanded, err := svc.Expand(ctx, ExpandParams{IDs: []string{search.Results[0].ExpandableID}})
    if err != nil { t.Fatal(err) }
    if expanded.Count != 1 { t.Fatalf("count = %d, want 1", expanded.Count) }
}
```

- [ ] **Step 6: Implement minimal expand API**

Create `search_expand.go` with `ExpandParams`, `ExpandResult`, and `Service.Expand`. Support `conclusion:<id>` first. Return `ErrObservationNotFound` or a new typed error for unsupported/missing IDs.

- [ ] **Step 7: Full verification and commit**

```bash
go test ./...
git add search_expand.go search_expand_test.go types.go service.go goncho_public_tools.go
git commit -m "feat: add progressive search expansion"
```

---

## Task 5: Project Profile and Context Boot Pack

**Capability copied from agentmemory:** `memory_profile` returns top concepts, files, conventions, common errors, recent activity, session counts, and observation counts.

**Goncho interpretation:** Project profile is orientation, not authority. It can be injected into context packs, but every claim should remain citeable or marked heuristic.

**Files:**

- Create: `project_profile.go`
- Create: `project_profile_test.go`
- Modify: `types.go` if shared structs should be exported there.
- Modify: `service.go` only if `Context` needs to include profile output.

- [ ] **Step 1: Write failing profile test**

Add `project_profile_test.go`:

```go
package goncho

import (
    "context"
    "testing"

    "github.com/TrebuchetDynamics/goncho/memory"
)

func TestProjectProfileCountsConceptsAndFilesFromEvidence(t *testing.T) {
    ctx := context.Background()
    store, err := memory.OpenSqlite(t.TempDir()+"/memory.db", 0, nil)
    if err != nil { t.Fatal(err) }
    defer store.Close(ctx)
    if err := RunMigrations(store.DB()); err != nil { t.Fatal(err) }
    svc := NewService(store.DB(), Config{WorkspaceID: "ws", ObserverPeerID: "agent"}, nil)
    _, err = svc.Observe(ctx, ObservationParams{Kind: ObservationKindToolResult, Output: "SQLite migration failed in memory.go", Metadata: map[string]string{"file":"memory.go","concepts":"sqlite,migration"}})
    if err != nil { t.Fatal(err) }

    profile, err := svc.ProjectProfile(ctx, ProjectProfileParams{WorkspaceID: "ws", Refresh: true})
    if err != nil { t.Fatal(err) }
    if profile.ObservationCount != 1 { t.Fatalf("observation count = %d", profile.ObservationCount) }
    if len(profile.TopFiles) == 0 || profile.TopFiles[0].File != "memory.go" { t.Fatalf("top files = %+v", profile.TopFiles) }
    if len(profile.TopConcepts) == 0 || profile.TopConcepts[0].Concept != "sqlite" { t.Fatalf("top concepts = %+v", profile.TopConcepts) }
}
```

- [ ] **Step 2: Run narrow test and confirm RED**

```bash
go test ./... -run TestProjectProfileCountsConceptsAndFilesFromEvidence
```

Expected: missing `ProjectProfileParams` and `ProjectProfile` method.

- [ ] **Step 3: Implement profile structs and method**

Create `project_profile.go` with:

- `ProjectProfileParams`
- `ProjectProfileResult`
- `ProjectProfileCount`
- `ProjectFileCount`
- `Service.ProjectProfile`

Initial implementation can derive counts from observation metadata and simple lower-cased output tokenization. Keep it deterministic and local.

- [ ] **Step 4: Add context boot-pack test**

Write a test proving `Service.Context` includes project profile orientation when a new `ContextParams.IncludeProjectProfile` bool is true.

- [ ] **Step 5: Add context parameter and output field**

Modify `types.go`:

```go
IncludeProjectProfile *bool `json:"include_project_profile,omitempty"`
```

to `ContextParams`, and:

```go
ProjectProfile *ProjectProfileResult `json:"project_profile,omitempty"`
```

to `ContextResult`.

Modify `Service.Context` to call `ProjectProfile` when requested.

- [ ] **Step 6: Full verification and commit**

```bash
go test ./...
git add project_profile.go project_profile_test.go types.go service.go
git commit -m "feat: add project profile boot context"
```

---

## Task 6: Relations and Query Graph

**Capability copied from agentmemory:** `memory_relations` and `memory_graph_query` for graph traversal.

**Goncho interpretation:** Relations connect evidence, claims, beliefs, files, sessions, and review items. Edges must carry relation type, confidence, source, and timestamps.

**Files:**

- Create: `relations.go`
- Create: `relations_test.go`
- Add migration file if current migration pattern requires a schema version.

- [ ] **Step 1: Write failing relation create/query test**

Add `relations_test.go`:

```go
package goncho

import (
    "context"
    "testing"

    "github.com/TrebuchetDynamics/goncho/memory"
)

func TestRelationsQueryReturnsDirectEdges(t *testing.T) {
    ctx := context.Background()
    store, err := memory.OpenSqlite(t.TempDir()+"/memory.db", 0, nil)
    if err != nil { t.Fatal(err) }
    defer store.Close(ctx)
    if err := RunMigrations(store.DB()); err != nil { t.Fatal(err) }
    svc := NewService(store.DB(), Config{WorkspaceID: "ws", ObserverPeerID: "agent"}, nil)

    rel, err := svc.CreateRelation(ctx, RelationCreateParams{SourceID: "claim:1", TargetID: "obs:1", Type: RelationSupports, Confidence: 0.9})
    if err != nil { t.Fatal(err) }
    got, err := svc.Relations(ctx, RelationQuery{SourceID: rel.SourceID, MaxHops: 1, MinConfidence: 0.5})
    if err != nil { t.Fatal(err) }
    if got.Count != 1 { t.Fatalf("count = %d, want 1", got.Count) }
    if got.Edges[0].Type != RelationSupports { t.Fatalf("type = %q", got.Edges[0].Type) }
}
```

- [ ] **Step 2: Run narrow test and confirm RED**

Expected: relation types and methods undefined.

- [ ] **Step 3: Implement relation model**

Create `relations.go` with:

- `RelationType`
- constants: `supports`, `contradicts`, `supersedes`, `derived_from`, `mentions_file`, `caused_by`, `fixed_by`, `failed_attempt`, `owned_by`, `requires_review`
- `Relation`
- `RelationCreateParams`
- `RelationQuery`
- `RelationResult`
- `Service.CreateRelation`
- `Service.Relations`

Store in SQLite. If no relation table exists, add a migration using the repo's existing migration pattern. Query direct edges first; add BFS only after direct query passes.

- [ ] **Step 4: Add max-hop traversal test**

Add a test with `claim:1 -> obs:1 -> file:README.md`, query `MaxHops:2`, and expect two edges.

- [ ] **Step 5: Implement bounded BFS traversal**

Implement deterministic traversal with:

- max hop cap of 5
- min confidence filter
- no duplicate edges
- no infinite loops

- [ ] **Step 6: Full verification and commit**

```bash
go test ./...
git add relations.go relations_test.go <migration-files-if-any>
git commit -m "feat: add memory relation graph queries"
```

---

## Task 7: Verify and Provenance API

**Capability copied from agentmemory:** `memory_verify` traces provenance and audit trail for a memory or observation.

**Goncho interpretation:** Verification must explain what is known, why it is believed, current lifecycle state, citations, audit trail, and live warnings.

**Files:**

- Create: `verify.go`
- Create: `verify_test.go`
- Modify: `goncho_public_tools.go`
- Create: `verify_tool_test.go`

- [ ] **Step 1: Write failing verify test for observation provenance**

Add `verify_test.go`:

```go
package goncho

import (
    "context"
    "testing"

    "github.com/TrebuchetDynamics/goncho/memory"
)

func TestVerifyObservationReturnsAuditAndEvidence(t *testing.T) {
    ctx := context.Background()
    store, err := memory.OpenSqlite(t.TempDir()+"/memory.db", 0, nil)
    if err != nil { t.Fatal(err) }
    defer store.Close(ctx)
    if err := RunMigrations(store.DB()); err != nil { t.Fatal(err) }
    svc := NewService(store.DB(), Config{WorkspaceID: "ws", ObserverPeerID: "agent"}, nil)
    observed, err := svc.Observe(ctx, ObservationParams{Kind: ObservationKindCustom, Output: "verified evidence"})
    if err != nil { t.Fatal(err) }

    proof, err := svc.Verify(ctx, VerifyParams{ID: observed.Observation.ID})
    if err != nil { t.Fatal(err) }
    if proof.ID != observed.Observation.ID { t.Fatalf("id = %q", proof.ID) }
    if len(proof.AuditEvents) == 0 { t.Fatal("expected audit events") }
    if len(proof.Evidence) != 1 { t.Fatalf("evidence count = %d", len(proof.Evidence)) }
}
```

- [ ] **Step 2: Run narrow test and confirm RED**

Expected: `VerifyParams` and `Service.Verify` undefined.

- [ ] **Step 3: Implement minimal Verify API**

Create `verify.go` with:

- `VerifyParams`
- `VerificationProof`
- `VerificationWarning`
- `Service.Verify`

Initial support:

- observation ID lookup
- audit trail lookup using existing `AuditTrail`
- relation lookup if Task 6 is already merged
- warnings for missing object or redacted/quarantined evidence

- [ ] **Step 4: Add public `goncho_verify` tool test**

Add `verify_tool_test.go` to execute:

```json
{"id":"obs_..."}
```

Assert returned JSON contains `success:true`, `action:"verify"`, and the requested ID.

- [ ] **Step 5: Implement `GonchoVerifyTool`**

Modify `goncho_public_tools.go` with:

- `type GonchoVerifyTool struct{ svc *Service }`
- `NewGonchoVerifyTool`
- `Name() string { return "goncho_verify" }`
- schema requiring `id`
- `Execute` calling `svc.Verify`

- [ ] **Step 6: Full verification and commit**

```bash
go test ./...
git add verify.go verify_test.go verify_tool_test.go goncho_public_tools.go
git commit -m "feat: add provenance verification tool"
```

---

## Task 8: Export and Snapshot

**Capability copied from agentmemory:** JSON export/import and git-versioned snapshots.

**Goncho interpretation:** Export is portable local backup; snapshot is deterministic state capture. Neither should require network, cloud embeddings, or a hosted service.

**Files:**

- Create: `export_snapshot.go`
- Create: `export_snapshot_test.go`
- Add migration file only if persistent snapshot metadata is required.

- [ ] **Step 1: Write failing export test**

Add `export_snapshot_test.go`:

```go
package goncho

import (
    "context"
    "encoding/json"
    "testing"

    "github.com/TrebuchetDynamics/goncho/memory"
)

func TestExportMemoryIncludesObservationsAndMetadata(t *testing.T) {
    ctx := context.Background()
    store, err := memory.OpenSqlite(t.TempDir()+"/memory.db", 0, nil)
    if err != nil { t.Fatal(err) }
    defer store.Close(ctx)
    if err := RunMigrations(store.DB()); err != nil { t.Fatal(err) }
    svc := NewService(store.DB(), Config{WorkspaceID: "ws", ObserverPeerID: "agent"}, nil)
    if _, err := svc.Observe(ctx, ObservationParams{Kind: ObservationKindCustom, Output: "export me"}); err != nil { t.Fatal(err) }

    exported, err := svc.ExportMemory(ctx, ExportMemoryParams{WorkspaceID: "ws"})
    if err != nil { t.Fatal(err) }
    if exported.Version == "" { t.Fatal("missing export version") }
    if len(exported.Observations) != 1 { t.Fatalf("observations = %d, want 1", len(exported.Observations)) }
    if _, err := json.Marshal(exported); err != nil { t.Fatal(err) }
}
```

- [ ] **Step 2: Run narrow test and confirm RED**

Expected: missing `ExportMemoryParams` and `ExportMemory`.

- [ ] **Step 3: Implement export structs and method**

Create `export_snapshot.go` with:

- `ExportMemoryParams`
- `MemoryExport`
- `Service.ExportMemory`

Include:

- version string
- exported_at timestamp
- workspace_id
- observations
- audit events if available
- relations if Task 6 is merged
- review items if available

- [ ] **Step 4: Write failing snapshot test**

Add test:

```go
func TestCreateSnapshotReturnsDeterministicDigest(t *testing.T) {
    ctx := context.Background()
    store, err := memory.OpenSqlite(t.TempDir()+"/memory.db", 0, nil)
    if err != nil { t.Fatal(err) }
    defer store.Close(ctx)
    if err := RunMigrations(store.DB()); err != nil { t.Fatal(err) }
    svc := NewService(store.DB(), Config{WorkspaceID: "ws", ObserverPeerID: "agent"}, nil)
    if _, err := svc.Observe(ctx, ObservationParams{Kind: ObservationKindCustom, Output: "snapshot me"}); err != nil { t.Fatal(err) }

    snap, err := svc.CreateSnapshot(ctx, SnapshotParams{WorkspaceID: "ws", Label: "test"})
    if err != nil { t.Fatal(err) }
    if snap.Digest == "" { t.Fatal("missing digest") }
    if snap.Label != "test" { t.Fatalf("label = %q", snap.Label) }
}
```

- [ ] **Step 5: Implement snapshot digest**

Add:

- `SnapshotParams`
- `SnapshotResult`
- `Service.CreateSnapshot`

Implementation: call `ExportMemory`, marshal deterministic JSON, compute SHA-256 digest, return digest and counts. Persist snapshot metadata only if needed by a future rollback task.

- [ ] **Step 6: Full verification and commit**

```bash
go test ./...
git add export_snapshot.go export_snapshot_test.go
git commit -m "feat: add memory export and snapshot digest"
```

---

## README Update Task

Run this after any public API/tool task lands.

**Files:**

- Modify: `README.md`

- [ ] **Step 1: Add a short section under “What Goncho Provides Today”**

Add bullets only for shipped features. Do not mark planned features as done.

Example after Tasks 1, 3, and 7 are complete:

```markdown
| **Hook ingestion adapter** | Converts agent hook events into local evidence without trusting them as current beliefs. |
| **File history** | Finds prior evidence about specific files and returns it with stale-context warnings. |
| **Provenance verification** | Traces memories and observations back to evidence, audit events, and relations. |
```

- [ ] **Step 2: Add a small API snippet for each shipped public method**

Example for file history:

```go
history, err := svc.FileHistory(ctx, goncho.FileHistoryParams{
    Files: []string{"README.md"},
    Limit: 10,
})
```

- [ ] **Step 3: Verify docs build or text-only checks**

Run:

```bash
go test ./...
rg "Hook ingestion adapter|FileHistory|Verify" README.md
```

- [ ] **Step 4: Commit docs**

```bash
git add README.md
git commit -m "docs: document new Goncho memory interfaces"
```

---

## Verification Standard

For every task:

1. Run the narrow test and capture RED before implementation.
2. Implement minimal code.
3. Run the narrow test and capture GREEN.
4. Run `go test ./...`.
5. If full suite fails because of unrelated WIP, document:
   - command,
   - first failing package,
   - first error line,
   - files involved,
   - why the implemented slice is still locally verified.
6. Commit exactly one feature slice.

---

## Known Repo State at Plan Creation

At plan creation, `git status --short` showed unrelated uncommitted WIP:

```text
 M cmd/goncho-bench/main.go
 M cmd/goncho-bench/main_test.go
?? Makefile
?? artifacts/
?? cmd/goncho-bench/science.go
?? docs/benchmarks/failures/
?? docs/benchmarks/results/
?? scripts/
```

Do not mix those files into this feature series unless Juan explicitly assigns benchmark work.

---

## Rollout Order

Recommended order:

1. Hook ingestion adapter.
2. Privacy/redaction gate.
3. File history API and public tool.
4. Progressive search plus expand.
5. Project profile/context boot pack.
6. Relations/query graph.
7. Verify/provenance API and public tool.
8. Export/snapshot.

Reasoning:

- Tasks 1 and 2 harden ingestion before more recall surfaces are added.
- Task 3 gives immediate coding-agent value.
- Task 4 reduces context bloat before profile/graph features increase recall volume.
- Task 5 improves session boot orientation.
- Task 6 creates the substrate for richer provenance.
- Task 7 makes trust inspectable.
- Task 8 makes state portable and rollback-friendly.

---

## Self-Review

Spec coverage:

- Hook ingestion adapter: Task 1.
- Privacy/redaction gate: Task 2.
- File history API: Task 3.
- Progressive Search plus Expand: Task 4.
- Project profile/context boot pack: Task 5.
- Relations/query graph: Task 6.
- Verify/provenance API: Task 7.
- Export/snapshot: Task 8.
- Framing that agentmemory is an integration/interface donor while Goncho remains stricter: header, public surface boundary, task interpretations, rollout order.

Placeholder scan:

- No `TBD` placeholders.
- No “implement later” placeholders.
- Every implementation task has concrete files, method names, test names, commands, and expected outcomes.

Type consistency:

- Hook adapter uses existing `Service.Observe` and `ObservationParams`.
- File history uses existing `Service.ListObservations` and `ObservationQuery`.
- Search expansion extends existing `SearchParams`, `SearchHit`, and `Service.Search`.
- Project profile extends existing `ContextParams` and `ContextResult`.
- Verify uses existing `AuditTrail` and future/optional relations.

Execution handoff:

Plan complete and saved to `docs/superpowers/plans/2026-05-20-agentmemory-donor-integration.md`. Two execution options:

1. Subagent-Driven (recommended): dispatch a fresh subagent per task, review between tasks, fast iteration.
2. Inline Execution: execute tasks in this session using executing-plans, batch execution with checkpoints.

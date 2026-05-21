# Lifecycle Write Module Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move Goncho lifecycle write orchestration into an unexported lifecycle module without public behavior changes.

**Architecture:** Keep public `Service` as stable seam. Add `lifecycleModule` in package `goncho`; `Service.CreateMessages`, `Service.DeleteSession`, and `Service.DeleteWorkspace` delegate to it. Preserve SQL helpers in `sql.go`.

**Tech Stack:** Go 1.25, SQLite via `database/sql`, existing Goncho package tests.

---

## File Structure

- Create: `lifecycle_module.go`
  - owns `lifecycleModule`, `Service.lifecycle`, lifecycle create/delete orchestration, lock retry helpers, and transient lock detection.
- Modify: `service.go`
  - replace `Service.CreateMessages`, `Service.DeleteSession`, and `Service.DeleteWorkspace` bodies with delegation.
  - remove moved private lifecycle helpers from `service.go`.
- Test: existing tests only unless behavior drift is found.
  - `crud_invariants_test.go`
  - `service_test.go`
  - `chat_contract_test.go`
  - `memory_tools_test.go`
  - `streaming_chat_persistence_test.go`
  - `local_e2e_test.go`
  - `http/local_e2e_test.go`

## Task 1: Baseline lifecycle behavior

**Files:** none

- [ ] **Step 1: Run targeted lifecycle tests**

Run:

```bash
cd /home/xel/git/sages-openclaw/workspace-mineru/goncho
go test . -run 'Test.*Create|Test.*Delete|Test.*CRUD|Test.*Lifecycle|Test.*Streaming|Test.*LocalE2E' -count=1
```

Expected: PASS.

- [ ] **Step 2: Capture dirty baseline**

Run:

```bash
git status --short
```

Expected: existing unrelated dirty path may remain:

```text
 M docs/opensource-memory-systems/agentmemory
```

Do not edit or stage that path.

## Task 2: Add lifecycle module seam

**Files:**
- Create: `lifecycle_module.go`

- [ ] **Step 1: Create `lifecycle_module.go` skeleton**

Create exact starting file:

```go
package goncho

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type lifecycleModule struct {
	db             *sql.DB
	workspaceID    string
	maxMessageSize int
}

func (s *Service) lifecycle() lifecycleModule {
	return lifecycleModule{
		db:             s.db,
		workspaceID:    s.workspaceID,
		maxMessageSize: s.maxMessageSize,
	}
}
```

- [ ] **Step 2: Run compile check**

Run:

```bash
go test . -run '^$'
```

Expected: PASS. If imports are unused, keep only imports needed at this step; later tasks add the rest.

- [ ] **Step 3: Commit seam skeleton**

Run:

```bash
git add lifecycle_module.go
git commit -m "refactor: add lifecycle module seam"
```

Expected: commit succeeds. Do not stage `docs/opensource-memory-systems/agentmemory`.

## Task 3: Move CreateMessages orchestration

**Files:**
- Modify: `service.go`
- Modify: `lifecycle_module.go`

- [ ] **Step 1: Replace `Service.CreateMessages` with delegation**

In `service.go`, replace current `Service.CreateMessages` body with:

```go
func (s *Service) CreateMessages(ctx context.Context, params CreateMessagesParams) (CreateMessagesResult, error) {
	return s.lifecycle().CreateMessages(ctx, params)
}
```

- [ ] **Step 2: Add `lifecycleModule.CreateMessages`**

In `lifecycle_module.go`, add:

```go
func (l lifecycleModule) CreateMessages(ctx context.Context, params CreateMessagesParams) (CreateMessagesResult, error) {
```

Use exact old `Service.CreateMessages` body with receiver edit:

```text
s.createMessagesOnce(...) -> l.createMessagesOnce(...)
```

- [ ] **Step 3: Move private create helpers**

Move these from `service.go` to `lifecycle_module.go`:

```go
func (l lifecycleModule) createMessagesOnce(ctx context.Context, sessionKey string, inputs []CreateMessage) (CreateMessagesResult, error)
func waitCreateMessagesLockRetry(ctx context.Context, attempt int) error
func createMessagesLockRetryDelay(attempt int) time.Duration
func isTransientSQLiteLockError(err error) bool
```

For `createMessagesOnce`, receiver edits:

```text
s.db -> l.db
s.workspaceID -> l.workspaceID
s.maxMessageSize -> l.maxMessageSize
```

Pure helper funcs keep same names and bodies.

- [ ] **Step 4: Run create-message tests**

Run:

```bash
go test . -run 'Test.*Create|Test.*CRUD|Test.*Streaming|Test.*LocalE2E' -count=1
```

Expected: PASS.

- [ ] **Step 5: Run package test**

Run:

```bash
go test . -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit create extraction**

Run:

```bash
git add service.go lifecycle_module.go
git commit -m "refactor: move create-message lifecycle orchestration"
```

Expected: commit succeeds.

## Task 4: Move DeleteSession and DeleteWorkspace orchestration

**Files:**
- Modify: `service.go`
- Modify: `lifecycle_module.go`

- [ ] **Step 1: Replace `Service.DeleteSession` with delegation**

In `service.go`, replace current `Service.DeleteSession` body with:

```go
func (s *Service) DeleteSession(ctx context.Context, sessionKey string) (SessionDeletionResult, error) {
	return s.lifecycle().DeleteSession(ctx, sessionKey)
}
```

- [ ] **Step 2: Add `lifecycleModule.DeleteSession`**

In `lifecycle_module.go`, add:

```go
func (l lifecycleModule) DeleteSession(ctx context.Context, sessionKey string) (SessionDeletionResult, error) {
```

Use exact old `Service.DeleteSession` body with receiver edits:

```text
s.db -> l.db
s.workspaceID -> l.workspaceID
```

- [ ] **Step 3: Replace `Service.DeleteWorkspace` with delegation**

In `service.go`, replace current `Service.DeleteWorkspace` body with:

```go
func (s *Service) DeleteWorkspace(ctx context.Context) (WorkspaceDeletionResult, error) {
	return s.lifecycle().DeleteWorkspace(ctx)
}
```

- [ ] **Step 4: Add `lifecycleModule.DeleteWorkspace`**

In `lifecycle_module.go`, add:

```go
func (l lifecycleModule) DeleteWorkspace(ctx context.Context) (WorkspaceDeletionResult, error) {
```

Use exact old `Service.DeleteWorkspace` body with receiver edits:

```text
s.db -> l.db
s.workspaceID -> l.workspaceID
```

- [ ] **Step 5: Run delete tests**

Run:

```bash
go test . -run 'Test.*Delete|Test.*CRUD|Test.*Lifecycle' -count=1
```

Expected: PASS.

- [ ] **Step 6: Run package test**

Run:

```bash
go test . -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit delete extraction**

Run:

```bash
git add service.go lifecycle_module.go
git commit -m "refactor: move lifecycle deletion orchestration"
```

Expected: commit succeeds.

## Task 5: Full verification and push

**Files:** none unless import cleanup required.

- [ ] **Step 1: Run targeted lifecycle tests**

Run:

```bash
go test . -run 'Test.*Create|Test.*Delete|Test.*CRUD|Test.*Lifecycle|Test.*Streaming|Test.*LocalE2E' -count=1
```

Expected: PASS.

- [ ] **Step 2: Run full tests**

Run:

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 3: Run race tests**

Run:

```bash
go test -race ./...
```

Expected: PASS.

- [ ] **Step 4: Run vet**

Run:

```bash
go vet ./...
```

Expected: PASS.

- [ ] **Step 5: Check public interface unchanged**

Run:

```bash
git diff HEAD~2..HEAD -- service.go types.go | rg '^[-+]func \(s \*Service\) (CreateMessages|DeleteSession|DeleteWorkspace)|^[-+]type (CreateMessagesParams|CreateMessagesResult|SessionDeletionResult|WorkspaceDeletionResult)' || true
```

Expected: only `Service` method bodies changed; no exported method signature or exported type change.

- [ ] **Step 6: Push commits**

Run:

```bash
git push origin main
```

Expected: push succeeds.

- [ ] **Step 7: Final report**

Report:

```text
Repo: /home/xel/git/sages-openclaw/workspace-mineru/goncho
Branch: main
Commits:
- <hash> refactor: add lifecycle module seam
- <hash> refactor: move create-message lifecycle orchestration
- <hash> refactor: move lifecycle deletion orchestration
Validation:
- go test ./... => PASS
- go test -race ./... => PASS
- go vet ./... => PASS
Unrelated dirty path:
- docs/opensource-memory-systems/agentmemory
```

## Plan Self-Review

Spec coverage:

- Public `Service` methods stay stable: Tasks 3 and 4.
- Retry and transaction ordering preserved: exact-body move with receiver-only edits.
- SQL helpers stay in `sql.go`: no task moves SQL helpers.
- No schema/retry behavior change: no task edits constants or SQL schema.
- Full verification: Task 5.

Placeholder scan: no unfinished-marker terms or unspecified test steps.

Type consistency:

- `lifecycleModule` name matches spec.
- Method signatures match existing public params/results.
- Receiver-only edits listed explicitly.

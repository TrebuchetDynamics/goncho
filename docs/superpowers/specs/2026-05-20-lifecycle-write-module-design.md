# Lifecycle Write Module Design

## Goal

Deepen Goncho lifecycle writes without changing public behavior.

Public `Service.CreateMessages`, `Service.DeleteSession`, and `Service.DeleteWorkspace` stay stable. Their transaction, retry, and lifecycle deletion orchestration moves behind one unexported module.

## Current friction

`Service` still owns write orchestration after the retrieval extraction:

- `CreateMessages` handles lock retry, transient SQLite error detection, transaction setup, lifecycle message creation, and retry delay.
- `DeleteSession` handles session-key validation, transaction setup, lifecycle deletion, commit, and rollback.
- `DeleteWorkspace` handles transaction setup, lifecycle deletion, commit, and rollback.
- SQL helpers already hide row-level operations, but the public facade still owns write-ordering rules.

Deletion test: deleting these methods from `Service` would not remove complexity. Transaction/retry/delete ordering would reappear in callers or helper functions. Lifecycle writes deserve a deeper module.

## Chosen approach

Create an unexported lifecycle module in package `goncho`.

Suggested shape:

```go
type lifecycleModule struct {
    db             *sql.DB
    workspaceID    string
    maxMessageSize int
}

func (s *Service) lifecycle() lifecycleModule
func (l lifecycleModule) CreateMessages(ctx context.Context, params CreateMessagesParams) (CreateMessagesResult, error)
func (l lifecycleModule) DeleteSession(ctx context.Context, sessionKey string) (SessionDeletionResult, error)
func (l lifecycleModule) DeleteWorkspace(ctx context.Context) (WorkspaceDeletionResult, error)
```

This is not a new package. One SQLite adapter exists, so a package seam would be premature.

## Interface rules

The lifecycle module interface includes these invariants:

- `CreateMessages` requires a non-empty `session_key`.
- `CreateMessages` preserves current lock retry count and backoff.
- Transient SQLite lock detection remains identical.
- Message creation still happens inside one transaction.
- Failed message creation rolls back.
- Commit failure returns a wrapped commit error.
- `DeleteSession` requires a non-empty `session_key`.
- Session deletion still happens inside one transaction.
- Workspace deletion still happens inside one transaction.
- Rollback remains best-effort when commit did not occur.
- Public result types stay unchanged.

## Files involved

Primary:

- `service.go`
- `sql.go`

Likely new file:

- `lifecycle_module.go`

Tests already exercising behavior:

- `crud_invariants_test.go`
- `service_test.go`
- `chat_contract_test.go`
- `memory_tools_test.go`
- `streaming_chat_persistence_test.go`
- `local_e2e_test.go`
- `http/local_e2e_test.go`

## Non-goals

- No public `Service` interface change.
- No new exported package.
- No SQL helper move unless compiler/locality requires it.
- No schema change.
- No retry behavior change.
- No cleanup of unrelated dirty submodule state.

## Migration plan

1. Add `lifecycleModule` with fields copied from `Service` as needed.
2. Add `Service.lifecycle()` constructor.
3. Move `Service.CreateMessages` implementation into `lifecycleModule.CreateMessages`.
4. Move private create-message helpers if tied to lifecycle ordering:
   - `createMessagesOnce`
   - `waitCreateMessagesLockRetry`
   - `createMessagesLockRetryDelay`
   - `isTransientSQLiteLockError`
5. Make `Service.CreateMessages` delegate.
6. Move `Service.DeleteSession` implementation into `lifecycleModule.DeleteSession`.
7. Make `Service.DeleteSession` delegate.
8. Move `Service.DeleteWorkspace` implementation into `lifecycleModule.DeleteWorkspace`.
9. Make `Service.DeleteWorkspace` delegate.
10. Run verification.

## Testing

Required commands after implementation:

```bash
go test . -run 'Test.*Create|Test.*Delete|Test.*CRUD|Test.*Lifecycle|Test.*Streaming|Test.*LocalE2E' -count=1
go test ./...
go test -race ./...
go vet ./...
```

Expected outcome: tests pass without changed assertions. If assertions must change, stop and treat that as behavior drift.

## Success criteria

- `Service.CreateMessages` delegates instead of owning retry/transaction orchestration.
- `Service.DeleteSession` delegates instead of owning transaction orchestration.
- `Service.DeleteWorkspace` delegates instead of owning transaction orchestration.
- Lifecycle write ordering stays identical.
- Public types/signatures stay identical.
- Full Go tests, race tests, and vet pass.
- `git diff` shows no unrelated edits besides lifecycle-module extraction.

## Risks

- Moving retry helpers can accidentally alter lock retry behavior.
- Transaction rollback order could drift.
- Delete-session validation error text could drift.
- Tests may not pin every rollback edge.

Mitigation: extraction-first refactor. Preserve code text where possible. Add tests only if a lifecycle invariant is found unpinned.

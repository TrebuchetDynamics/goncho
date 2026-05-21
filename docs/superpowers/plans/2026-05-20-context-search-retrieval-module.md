# Context Search Retrieval Module Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deepen Goncho retrieval by moving `Service.Search` and `Service.Context` orchestration into an unexported retrieval module without public behavior changes.

**Architecture:** Keep public `Service` as the stable seam. Add `retrievalModule` as an internal deep module in package `goncho`; `Service.Search` and `Service.Context` delegate to it. Preserve SQL helpers and existing public result types.

**Tech Stack:** Go 1.25, SQLite via `database/sql`, existing Goncho package tests.

---

## File Structure

- Create: `retrieval_module.go`
  - owns `retrievalModule`, `Service.retrieval`, `retrievalModule.Search`, `retrievalModule.Context`, `retrievalModule.searchTurnFallback`, `retrievalModule.crossChatEvidenceMetadata`, and summary-refresh helpers if needed.
- Modify: `service.go`
  - replace `Service.Search` and `Service.Context` bodies with delegation.
  - remove moved helper methods from `Service` receiver.
  - keep shared pure helpers if still used by chat or nearby code.
- Test: existing tests only unless behavior gap appears.
  - `service_test.go`
  - `context_options_test.go`
  - `summary_context_test.go`
  - `cross_session_test.go`
  - `review_context_test.go`
  - `prompt_injection_quarantine_test.go`
  - `search_candidate_generation_test.go`
  - `search_rank_temporal_test.go`

## Task 1: Baseline behavior before refactor

**Files:** none

- [ ] **Step 1: Run targeted retrieval tests**

Run:

```bash
cd /home/xel/git/sages-openclaw/workspace-mineru/goncho
go test . -run 'Test.*Context|Test.*Search|Test.*Summary|Test.*Cross|Test.*Quarantine|Test.*Review' -count=1
```

Expected: PASS.

- [ ] **Step 2: Run candidate-generation regression**

Run:

```bash
go test . -run TestSearchCandidateGenerationKeepsOldStrongLexicalMatch -count=1
```

Expected: PASS.

- [ ] **Step 3: Capture dirty baseline**

Run:

```bash
git status --short
```

Expected: existing unrelated dirty path may remain:

```text
 M docs/opensource-memory-systems/agentmemory
```

Do not edit or stage that path.

## Task 2: Add retrieval module seam

**Files:**
- Create: `retrieval_module.go`
- Modify: `service.go`

- [ ] **Step 1: Create `retrieval_module.go` with module skeleton**

Create exact starting file:

```go
package goncho

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type retrievalModule struct {
	db              *sql.DB
	workspaceID     string
	observer        string
	recentLimit     int
	peerCardEnabled bool
	dreamEnabled    bool
	sessions        SessionDirectory
}

func (s *Service) retrieval() retrievalModule {
	return retrievalModule{
		db:              s.db,
		workspaceID:     s.workspaceID,
		observer:        s.observer,
		recentLimit:     s.recentLimit,
		peerCardEnabled: s.peerCardEnabled,
		dreamEnabled:    s.dreamEnabled,
		sessions:        s.sessions,
	}
}
```

- [ ] **Step 2: Run compile check**

Run:

```bash
go test . -run '^$'
```

Expected: PASS.

- [ ] **Step 3: Commit seam skeleton**

Run:

```bash
git add retrieval_module.go
git commit -m "refactor: add retrieval module seam"
```

Expected: commit succeeds. Do not stage `docs/opensource-memory-systems/agentmemory`.

## Task 3: Move Search orchestration

**Files:**
- Modify: `service.go`
- Modify: `retrieval_module.go`

- [ ] **Step 1: Replace `Service.Search` with delegation**

In `service.go`, replace the current `func (s *Service) Search(...)` body with:

```go
func (s *Service) Search(ctx context.Context, params SearchParams) (SearchResultSet, error) {
	return s.retrieval().Search(ctx, params)
}
```

- [ ] **Step 2: Add `retrievalModule.Search`**

In `retrieval_module.go`, add a method named:

```go
func (r retrievalModule) Search(ctx context.Context, params SearchParams) (SearchResultSet, error) {
```

Use the exact old `Service.Search` implementation body, with these receiver edits only:

```text
s.db          -> r.db
s.workspaceID -> r.workspaceID
s.observer    -> r.observer
s.searchTurnFallback(...) -> r.searchTurnFallback(...)
```

The returned `SearchResultSet` must still populate:

```go
SearchResultSet{
	WorkspaceID:   r.workspaceID,
	ProfileID:     profileID,
	Peer:          peer,
	Query:         params.Query,
	ScopeEvidence: scopeEvidence,
	Results:       results,
}
```

- [ ] **Step 3: Move `searchTurnFallback` receiver**

Move existing `func (s *Service) searchTurnFallback(...)` from `service.go` to `retrieval_module.go` and change only:

```go
func (r retrievalModule) searchTurnFallback(ctx context.Context, params SearchParams, compiled compiledSearchFilter, limit int) (turnFallbackResult, error)
```

Receiver edits inside body:

```text
s.sessions -> r.sessions
s.db -> r.db
s.crossChatEvidenceMetadata(...) -> r.crossChatEvidenceMetadata(...)
```

- [ ] **Step 4: Move `crossChatEvidenceMetadata` receiver**

Move existing `func (s *Service) crossChatEvidenceMetadata(...)` to `retrieval_module.go` and change only:

```go
func (r retrievalModule) crossChatEvidenceMetadata(ctx context.Context, userID, currentKey string, metas []SessionMetadata) ([]SessionMetadata, error)
```

Receiver edit inside body:

```text
s.sessions -> r.sessions
```

- [ ] **Step 5: Run search tests**

Run:

```bash
go test . -run 'Test.*Search|Test.*Cross' -count=1
```

Expected: PASS.

- [ ] **Step 6: Run full package compile/test**

Run:

```bash
go test . -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit Search extraction**

Run:

```bash
git add service.go retrieval_module.go
git commit -m "refactor: move search orchestration into retrieval module"
```

Expected: commit succeeds. No public type/signature changes.

## Task 4: Move Context orchestration

**Files:**
- Modify: `service.go`
- Modify: `retrieval_module.go`

- [ ] **Step 1: Replace `Service.Context` with delegation**

In `service.go`, replace current `func (s *Service) Context(...)` body with:

```go
func (s *Service) Context(ctx context.Context, params ContextParams) (ContextResult, error) {
	return s.retrieval().Context(ctx, params)
}
```

- [ ] **Step 2: Add `retrievalModule.Context`**

In `retrieval_module.go`, add:

```go
func (r retrievalModule) Context(ctx context.Context, params ContextParams) (ContextResult, error) {
```

Use exact old `Service.Context` implementation body, with these receiver edits only:

```text
s.observer -> r.observer
s.db -> r.db
s.workspaceID -> r.workspaceID
s.reviewContextUnavailableEvidence(...) -> reviewContextUnavailableEvidence(ctx, r.db, r.workspaceID, r.observer, peer)
s.dreamContextUnavailableEvidence(...) -> dreamContextUnavailableEvidence(ctx, r.db, r.workspaceID, r.observer, peer)
s.Search(...) -> r.Search(...)
s.refreshSessionSummaries(...) -> r.refreshSessionSummaries(...)
s.recentLimit -> r.recentLimit
```

If direct conversion of `reviewContextUnavailableEvidence` or `dreamContextUnavailableEvidence` creates broad edits, keep tiny `Service` helper delegation instead and defer those two helpers. Do not change behavior.

- [ ] **Step 3: Move summary refresh helpers**

Move existing summary-refresh receiver helpers to `retrieval_module.go`:

```go
func (r retrievalModule) refreshSessionSummaries(ctx context.Context, sessionKey string) (int, error)
func (r retrievalModule) refreshSessionSummarySlot(ctx context.Context, sessionKey, summaryType string, cadence, turnCount int) error
```

Receiver edits:

```text
s.db -> r.db
s.workspaceID -> r.workspaceID
s.refreshSessionSummarySlot(...) -> r.refreshSessionSummarySlot(...)
```

- [ ] **Step 4: Run context tests**

Run:

```bash
go test . -run 'Test.*Context|Test.*Summary|Test.*Quarantine|Test.*Review' -count=1
```

Expected: PASS.

- [ ] **Step 5: Run full package test**

Run:

```bash
go test . -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit Context extraction**

Run:

```bash
git add service.go retrieval_module.go
git commit -m "refactor: move context orchestration into retrieval module"
```

Expected: commit succeeds. No public type/signature changes.

## Task 5: Verify no behavior drift

**Files:** none unless compiler forces import cleanup.

- [ ] **Step 1: Run targeted suite**

Run:

```bash
go test . -run 'Test.*Context|Test.*Search|Test.*Summary|Test.*Cross|Test.*Quarantine|Test.*Review' -count=1
```

Expected: PASS.

- [ ] **Step 2: Run candidate regression**

Run:

```bash
go test . -run TestSearchCandidateGenerationKeepsOldStrongLexicalMatch -count=1
```

Expected: PASS.

- [ ] **Step 3: Run full tests**

Run:

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 4: Run race tests**

Run:

```bash
go test -race ./...
```

Expected: PASS.

- [ ] **Step 5: Run vet**

Run:

```bash
go vet ./...
```

Expected: PASS.

- [ ] **Step 6: Check public interface unchanged**

Run:

```bash
git diff HEAD~2..HEAD -- types.go memory.go service.go | rg '^[-+]func \(s \*Service\) (Search|Context)|^[-+]type (SearchParams|ContextParams|SearchResultSet|ContextResult)' || true
```

Expected: only `Service.Search` and `Service.Context` body lines changed; no exported type or method signature changes.

- [ ] **Step 7: Check unrelated dirty path remains unstaged**

Run:

```bash
git status --short
```

Expected: refactor commits are clean except pre-existing dirty path if still present:

```text
 M docs/opensource-memory-systems/agentmemory
```

## Task 6: Push final refactor commits

**Files:** none

- [ ] **Step 1: Push branch**

Run:

```bash
git push origin main
```

Expected: push succeeds.

- [ ] **Step 2: Final report**

Report exact evidence:

```text
Repo: /home/xel/git/sages-openclaw/workspace-mineru/goncho
Branch: main
Commits:
- <hash> refactor: add retrieval module seam
- <hash> refactor: move search orchestration into retrieval module
- <hash> refactor: move context orchestration into retrieval module
Validation:
- go test ./... => PASS
- go test -race ./... => PASS
- go vet ./... => PASS
Unrelated dirty path:
- docs/opensource-memory-systems/agentmemory
```

## Plan Self-Review

Spec coverage:

- Public `Service.Context` and `Service.Search` stay stable: Task 3 and Task 4.
- Retrieval ordering preserved: move exact old bodies; receiver-only edits.
- SQL helpers stay in place: no task moves SQL helpers.
- No ranking/benchmark optimization: no task touches `search_rank.go` or benchmark code.
- Full verification: Task 5.

Placeholder scan: no unfinished-marker terms or unspecified test steps.

Type consistency:

- `retrievalModule` name matches spec.
- Method signatures match existing public params/results.
- Receiver-only edits listed explicitly.

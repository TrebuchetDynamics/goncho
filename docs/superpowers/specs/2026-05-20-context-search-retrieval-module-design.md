# Context + Search Retrieval Module Design

## Goal

Deepen Goncho context/search retrieval without changing public behavior.

The public `Service.Context` and `Service.Search` interfaces stay stable. Their orchestration moves behind one internal retrieval module so retrieval ordering, fallback behavior, unavailable evidence, summary windows, and token budgets have better locality.

## Current friction

`Service.Context` and `Service.Search` are shallow at the implementation seam:

- `Service.Context` knows profile-card reads, unavailable evidence collection, review/dream/quarantine checks, search invocation, summary refresh, recent-turn loading, and token-budget splitting.
- `Service.Search` knows filter compilation, source merging, conclusion lookup, empty-query fallback, turn fallback, token limiting, and scope evidence.
- Tests cross the public `Service` interface, but bugs in retrieval ordering can require reading `service.go`, `sql.go`, `filter.go`, `review.go`, `dream_scheduler.go`, and `quarantine.go` together.

Deletion test: deleting the current orchestration would not remove complexity. It would reappear across callers or future retrieval features. That means retrieval deserves a deeper module.

## Chosen approach

Create an unexported retrieval module near the service layer.

Suggested shape:

```go
type retrievalModule struct {
    db              *sql.DB
    workspaceID     string
    observer        string
    recentLimit     int
    peerCardEnabled bool
    dreamEnabled    bool
}

func (r retrievalModule) Search(ctx context.Context, params SearchParams) (SearchResultSet, error)
func (r retrievalModule) Context(ctx context.Context, params ContextParams) (ContextResult, error)
```

`Service` constructs this module internally and delegates from public methods.

This is not a new package yet. A package seam would be premature because there is only one adapter and no public need for separate import paths.

## Interface rules

The retrieval module interface includes more than method signatures:

- `Peer` is required for both search and context.
- Search filter compilation still happens before SQL lookup.
- Deny-all filters return empty results, not errors.
- Conclusion search still runs before turn fallback.
- Empty conclusion result with non-empty query may retry conclusion search with empty query.
- Turn fallback remains the fallback after conclusion search is exhausted.
- Token limiting happens after candidate retrieval.
- Context unavailable evidence includes requested-capability evidence, dream evidence, review evidence, and quarantine evidence in current behavior-compatible order.
- Session summaries are refreshed before selecting summary/recent-turn context.
- Public result types stay unchanged.

## Files involved

Primary:

- `service.go`
- `sql.go`
- `filter.go`
- `quarantine.go`
- `review.go`
- `dream_scheduler.go`

Likely new file:

- `retrieval_module.go`

Tests already exercising behavior:

- `service_test.go`
- `context_options_test.go`
- `summary_context_test.go`
- `cross_session_test.go`
- `review_context_test.go`
- `prompt_injection_quarantine_test.go`
- `search_candidate_generation_test.go`
- `search_rank_temporal_test.go`

## Non-goals

- No public `Service` interface change.
- No new exported package.
- No ranking change.
- No benchmark optimization.
- No SQLite adapter abstraction.
- No broad file reorganization.
- No cleanup of unrelated dirty submodule state.

## Migration plan

1. Add `retrievalModule` with fields copied from `Service` as needed.
2. Move `Service.Search` implementation into `retrievalModule.Search`.
3. Make `Service.Search` delegate to `s.retrieval().Search(...)`.
4. Move `Service.Context` implementation into `retrievalModule.Context`.
5. Make `Service.Context` delegate to `s.retrieval().Context(...)`.
6. Move private helper methods only when required by compiler or locality:
   - `searchTurnFallback`
   - `crossChatEvidenceMetadata`
   - maybe summary/context helpers if directly tied to retrieval orchestration
7. Keep SQL helpers where they are.
8. Run full verification.

## Testing

Required commands after implementation:

```bash
go test ./...
go test -race ./...
go vet ./...
```

Targeted commands during extraction:

```bash
go test . -run 'Test.*Context|Test.*Search|Test.*Summary|Test.*Cross|Test.*Quarantine|Test.*Review' -count=1
go test . -run TestSearchCandidateGenerationKeepsOldStrongLexicalMatch -count=1
```

Expected outcome: tests pass without changed assertions. If assertions must change, stop and treat that as behavior drift.

## Success criteria

- `Service.Context` delegates instead of owning full orchestration.
- `Service.Search` delegates instead of owning full orchestration.
- Retrieval ordering stays identical.
- Public types/signatures stay identical.
- Full Go tests, race tests, and vet pass.
- `git diff` shows no unrelated edits besides this retrieval-module extraction.

## Risks

- Moving methods can accidentally change helper receiver access.
- Context unavailable evidence order could drift.
- Search fallback ordering could drift.
- Tests may not pin every ordering invariant.

Mitigation: extraction-first refactor. Preserve code text where possible. Add tests only if a behavior invariant is found unpinned.

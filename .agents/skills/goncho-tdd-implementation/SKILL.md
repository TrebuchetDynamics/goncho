---
name: goncho-tdd-implementation
description: Implement Goncho roadmap features with red-green-refactor. Use when changing Goncho behavior from TODO.md, README roadmap, docs/superpowers plans, or memory-system metaanalysis gaps.
---

# Goncho TDD Implementation

## Quick start

Identify the smallest user-visible contract, then write the failing test before production code.

```text
roadmap gap -> RED contract test -> minimal code -> GREEN -> go test ./... -> docs/status -> commit
```

## Workflow

1. **Locate the slice**
   - Name the TODO/README/docs/metaanalysis capability.
   - Inspect current code/tests before asking questions the repo can answer.
   - State what is explicitly out of scope.
2. **RED**
   - Add one focused test proving missing behavior.
   - Run the narrow test and confirm it fails for the expected reason.
3. **GREEN**
   - Implement the smallest behavior that satisfies the test.
   - Run the narrow test until it passes.
4. **Refactor and verify**
   - Refactor only while tests stay green.
   - Run `go test ./...`; add docs/status updates for public behavior.
5. **Closeout**
   - Summarize the slice, tests, docs, and remaining gaps.

## Test target guide

| Capability type | Preferred proof |
| --- | --- |
| Public API behavior | package or service `*_test.go` |
| Storage contract | package-specific storage test |
| HTTP/server behavior | `http/*_test.go` or `cmd/goncho-server/*_test.go` |
| Tool compatibility | `memory_tools_test.go`, `goncho_public_tools_test.go`, host integration tests |
| Recall quality | `recall_*_test.go`, benchmark harness tests |
| Lifecycle/trust | focused service test plus context/search/review proof |
| Docs/release metadata | guard test plus rendered/build smoke |

## Skill contract

### Entry protocol
- Trivial: if the change is docs/test-only and scoped, proceed with a narrow validation plan.
- Medium ambiguity: propose the smallest slice and ask only the missing owner decision.
- High ambiguity/risk: stop before broad rewrites, schema migrations, destructive operations, or behavior changes without a testable contract.

### Topology check
- State/ownership: which package/API owns the behavior and data.
- Feedback/validation: exact narrow test, full test, and docs guard if public.
- Blast radius: public API, migrations, benchmark artifacts, adapters, and host contracts.
- Timing/ordering: migrations, background jobs, replay/import order, and external side effects.

### Verification gate
Done requires:
- expected RED output captured or described,
- narrow GREEN test passes,
- `go test ./...` passes or blocker output is reported,
- public docs/status updated when behavior changes.

### Red lines
- No production behavior before a failing test.
- No broad rewrites hidden inside a feature slice.
- No schema or metadata fields that are not consumed by retrieval, lifecycle, context, review, audit, or tools.
- No regenerating benchmark/full-run artifacts unless explicitly in scope.
- No staging/committing unrelated local edits.

### Output contract
End with: slice name, tests added/changed, commands run, files changed, and remaining gaps/blockers.

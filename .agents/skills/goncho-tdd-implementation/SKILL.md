---
name: goncho-tdd-implementation
description: Use when implementing any Goncho feature from the memory-systems metaanalysis, feature matrix, README roadmap, or docs/superpowers plans.
---

# Goncho TDD Implementation

## Core Rule

No Goncho production code without a failing test first.

Every feature from `docs/opensource-memory-systems/METAANALYSIS-MEMORY-SYSTEMS.md` must be implemented as a small vertical slice:

```text
matrix gap -> failing contract test -> minimal code -> full go test ./... -> commit
```

## Required Inputs

Before coding, identify:

- the metaanalysis capability being implemented,
- the current implementation status,
- the public API or internal contract that should prove it,
- the smallest test that fails for the right reason.

## Quick Reference

| Step | Required evidence |
| --- | --- |
| Pick slice | Metaanalysis/roadmap gap and current implementation status |
| RED | Narrow test fails for the expected missing behavior |
| GREEN | Minimal implementation makes the narrow test pass |
| Full verify | `go test ./...` passes or blocker is documented |
| Docs/status | README/docs updated if public behavior changed |
| Commit | One feature slice named in commit message |

## TDD Loop

1. Add or update a test first.
2. Run the narrow test and confirm RED.
3. Implement the smallest code that makes it pass.
4. Run the narrow test and confirm GREEN.
5. Run `go test ./...`.
6. Refactor only while tests stay green.
7. Commit one feature slice.

## Test Targets

Prefer tests in this order:

| Capability type | Preferred test |
| --- | --- |
| Public API behavior | root package `*_test.go` |
| Storage contract | package-specific storage test |
| HTTP compatibility | `http/*_test.go` |
| Tool compatibility | `memory_tools_test.go`, `host_integration_test.go` |
| Recall quality | `recall_*_test.go`, `proof_matrix_test.go` |
| Lifecycle behavior | focused unit test plus integration test |

## Done Criteria

A feature is done only when:

- test failed before implementation,
- test passes after implementation,
- `go test ./...` passes,
- docs or README status are updated if public behavior changed,
- commit message names the feature slice.

## Common Mistakes

| Mistake | Fix |
| --- | --- |
| Writing schema or structs before a failing behavior test | Delete/ignore the draft and start from RED |
| Testing only storage for a user-visible feature | Add API, search, context, tool, or lifecycle behavior proof |
| Claiming a phase complete from one internal piece | Report the exact slice and remaining gaps |
| Letting refactor expand behavior | Keep refactors green and add a new RED for new behavior |
| Ignoring unrelated failing tests | Document blocker with exact output before claiming partial completion |

## Avoid

- broad rewrites,
- hidden behavior without tests,
- adding metadata fields that are not used by retrieval or lifecycle logic,
- claiming a metaanalysis phase is complete from schema alone.

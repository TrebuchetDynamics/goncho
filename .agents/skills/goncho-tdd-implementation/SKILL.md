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

## Avoid

- broad rewrites,
- hidden behavior without tests,
- adding metadata fields that are not used by retrieval or lifecycle logic,
- claiming a metaanalysis phase is complete from schema alone.

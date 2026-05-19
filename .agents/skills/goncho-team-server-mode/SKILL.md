---
name: goncho-team-server-mode
description: Use when implementing Goncho team memory, server mode, PostgreSQL adapters, ACLs, roles, import/export, or multi-user governance.
---

# Goncho Team and Server Mode

## Goal

Add shared memory without breaking the local-first solo workflow.

## Required TDD Shape

Use `goncho-tdd-implementation` first. Every team/server feature must prove isolation and compatibility:

- SQLite local mode still works,
- server/PostgreSQL mode preserves public contracts,
- workspace/user/team ACLs prevent leakage,
- audit trails identify actor and scope,
- import/export preserves provenance and lifecycle state.

## Minimal Contract Examples

Good tests:

- `TestTeamMemoryDoesNotLeakAcrossWorkspaces`
- `TestPostgresAdapterMatchesSQLiteMemoryContract`
- `TestRoleCannotReadPrivatePeerMemory`
- `TestExportImportPreservesAuditAndSupersession`

## Design Rules

- Keep storage behind interfaces with SQLite as the reference behavior.
- Add adapter conformance tests before adapter code.
- Treat ACL denial as observable evidence, not a silent empty result when possible.
- Server mode must not require cloud services for local mode.
- Role and actor metadata must be audit-visible.

## Done Criteria

- local SQLite tests pass unchanged,
- adapter conformance tests pass,
- ACL tests cover allow and deny paths,
- import/export round trips scoped memory,
- `go test ./...` passes.

## Avoid

- making PostgreSQL required for basic use,
- adding roles without enforcement tests,
- changing public Honcho/MCP contracts for server mode.

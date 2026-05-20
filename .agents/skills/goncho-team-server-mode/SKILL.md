---
name: goncho-team-server-mode
description: Use when implementing Goncho team memory, server mode, PostgreSQL adapters, ACLs, roles, import/export, or multi-user governance.
---

# Goncho Team and Server Mode

## Goal

Add shared memory without breaking the local-first solo workflow.

## Required TDD Shape

**REQUIRED SUB-SKILL:** Use `goncho-tdd-implementation` first. Every team/server feature must prove isolation and compatibility:

- SQLite local mode still works,
- server/PostgreSQL mode preserves public contracts,
- workspace/user/team ACLs prevent leakage,
- audit trails identify actor and scope,
- import/export preserves provenance and lifecycle state.

## Quick Reference

| Need | Prove with |
| --- | --- |
| Local-first compatibility | Existing SQLite tests pass unchanged |
| Adapter parity | PostgreSQL adapter passes SQLite memory contract tests |
| ACL isolation | Allow and deny paths are observable and tested |
| Auditability | Actor, scope, role, and decision are visible in audit output |
| Import/export | Round trip preserves provenance, lifecycle, and scope |
| Server mode | Public contracts work without requiring cloud for local mode |

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

## Common Mistakes

| Mistake | Fix |
| --- | --- |
| Making PostgreSQL required for normal local use | Keep SQLite as reference and local default |
| Adding roles without enforcement | Write allow and deny ACL tests before role code |
| Treating denied access as indistinguishable from no results | Return observable denial evidence where safe |
| Changing public contracts for server mode | Preserve API/tool compatibility and add adapter conformance tests |
| Dropping provenance during import/export | Assert actor, scope, lifecycle, and evidence survive round trip |

## Avoid

- making PostgreSQL required for basic use,
- adding roles without enforcement tests,
- changing public Goncho/MCP contracts for server mode.

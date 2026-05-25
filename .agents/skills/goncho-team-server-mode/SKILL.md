---
name: goncho-team-server-mode
description: Implement Goncho shared/team/server behavior. Use when working on server mode, team memory, PostgreSQL adapters, ACLs, roles, leases, receipts, team feeds, import/export, or multi-user governance.
---

# Goncho Team and Server Mode

## Quick start

Load `goncho-tdd-implementation`, then prove shared behavior without weakening local-first SQLite mode.

## Workflow

1. **Choose one shared-mode contract**
   - SQLite local compatibility.
   - Server auth/bind safety.
   - Workspace/profile/team ACL allow and deny behavior.
   - PostgreSQL adapter parity.
   - Action leases, receipts, or feed auditability.
   - Import/export provenance and lifecycle preservation.
2. **Write the contract test**
   - Good names: `TestTeamMemoryDoesNotLeakAcrossWorkspaces`, `TestPostgresAdapterMatchesSQLiteMemoryContract`, `TestRoleCannotReadPrivatePeerMemory`, `TestExportImportPreservesAuditAndSupersession`.
3. **Implement minimally**
   - Keep SQLite as reference behavior and local default.
   - Put storage behind interfaces before adding adapters.
   - Make ACL denials observable where safe.
   - Preserve public service/tool contracts.
4. **Verify**
   - Run narrow server/team tests, local SQLite tests, then `go test ./...`.

## Design rules

- Shared mode must not require cloud services for solo/local use.
- ACL decisions need actor, scope, role, decision, and audit evidence.
- Adapter parity requires conformance tests, not optimistic assumptions.
- Import/export must preserve provenance, lifecycle, scope, and review/audit state.

## Skill contract

### Entry protocol
- Trivial: answer server/team design questions using current docs/tests.
- Medium ambiguity: propose the smallest isolation or adapter-parity slice and ask only the missing policy decision.
- High ambiguity/risk: stop before network exposure, auth weakening, migrations, or multi-user data access changes without explicit scope.

### Topology check
- State/ownership: SQLite reference, server runtime, adapter interfaces, ACL policy, audit log.
- Feedback/validation: allow path, deny path, local compatibility, and adapter parity where relevant.
- Blast radius: local-first behavior, public APIs/tools, import/export, server bind/auth, data leakage.
- Timing/ordering: leases, receipts, pagination, migrations, concurrent writes, backfills.

### Verification gate
Done requires unchanged SQLite local behavior, tested allow/deny ACL paths, audit visibility, import/export parity where touched, and `go test ./...` pass or blocker output.

### Red lines
- Do not make PostgreSQL or server mode required for basic local use.
- Do not expose non-loopback server binds without explicit auth/token requirements.
- Do not add roles without enforcement tests.
- Do not make denied access indistinguishable from no data when observable denial is safe.
- Do not change public contracts for server mode without compatibility tests.

### Output contract
End with: shared-mode contract covered, isolation evidence, audit fields, validation commands, and remaining deployment/security risks.

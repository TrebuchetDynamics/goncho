# Goncho Server Mode Threat Model

Status: requirements-only; server-mode authorization contracts are documented before shared/team features are enabled.

## Scope

Server mode means running Goncho as a shared HTTP/MCP service for more than one local agent, profile, or team member. Embedded SQLite/local-first mode remains the reference path and must keep working without network auth, PostgreSQL, Docker, or cloud services.

## Assets and trust boundaries

- Memory content: observations, conclusions, slots, review items, snapshots, image refs, vector refs, and portable exports.
- Identity and scope: actor IDs, peer IDs, profiles, workspaces, sessions, and team membership.
- Governance metadata: audit rows, review decisions, stale/superseded state, retention archives, import/export manifests, and provider diagnostics.
- Admin surfaces: config, migration, backup/export/restore, retention apply, connector apply, and future team/lease/signal operations.
- Network boundary: loopback is trusted for solo local use; any non-loopback bind is shared-server mode and requires explicit server authentication.

## Required controls before shared/team enablement

1. **Auth**: default `serve` bind stays loopback-only. Non-loopback binds require an explicit server auth token or stronger future auth scheme. Unauthenticated non-loopback binds fail closed.
2. **Profiles**: every authenticated actor is bound to one or more profiles; private profile memory is denied by default.
3. **Workspaces**: workspace membership gates recall, viewer, review, import/export, retention, action, lease, and signal APIs.
4. **Audit**: allow and deny decisions record actor, role, workspace/profile scope, operation, decision, and reason without leaking protected memory content.
5. **Backup**: backup/export/restore uses snapshot manifests with checksums, provenance, lifecycle state, review state, and scope metadata.
6. **Retention**: retention preview/apply remains policy-scoped and audit-visible; shared mode cannot silently delete or expose another workspace/profile's archived memory.
7. **Admin operations**: migrations, connector apply, retention apply, import apply, backup restore, and auth/role changes require admin role and explicit preview/apply separation where destructive or externally mutating.

## Roles

- `admin`: manages auth, roles, workspace membership, migrations, backup/restore, retention apply, and connector apply.
- `operator`: runs health/doctor, non-destructive previews, eval gates, and approved maintenance within assigned workspaces.
- `reader`: uses recall/viewer/export on authorized workspaces and profiles only.

## PostgreSQL adapter plan

SQLite remains the reference implementation. A PostgreSQL adapter for team/shared deployments must land behind adapter interfaces with conformance tests proving parity with SQLite memory contracts, migrations, lifecycle/review state, scoped import/export, and ACL allow/deny behavior.

## Non-goals

- No P2P mesh sync before server mode is secure and boring.
- No hosted/cloud dependency for local mode.
- No role names without tested authorization decisions.

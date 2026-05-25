# Slack and Discord Connector Plan

Status: plan-after-server-acl

Slack and Discord support is intentionally planned after server-mode ACLs and retention are explicit. Team chats can contain sensitive multi-user data, so connector work must wait for workspace/profile authorization, role-aware audit, and retention previews.

## Scope

- Slack: selected workspaces/channels/threads only.
- Discord: selected guilds/channels/threads only.
- Team chats become scoped observations with channel/thread/message IDs, author identity mapping, timestamps, redaction summaries, and source backlinks.

## Required controls

- Server-mode ACLs must gate every workspace/profile authorization decision.
- Retention policy must be explicit before importing team chats.
- Preview channel allowlists, estimated counts, and sample redacted records before writing.
- Backfill must be bounded by channel, since/until, max messages, and rate-limit budget.
- Read receipts or imported reactions must not imply user consent to broaden memory scope.
- Audit entries must record actor, role, workspace, profile, channel, decision, and reason for allow/deny paths.

## Non-goals

- No Slack/Discord bot posting in the first connector.
- No private DM import without a separate threat model.
- No hosted sync service requirement for local Goncho.

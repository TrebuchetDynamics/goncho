# GitHub Connector Plan

Status: plan

A future GitHub connector should import issues, pull requests, discussions, review comments, and issue comments as scoped observations. It must be preview-first and local-first: no repository writes, no webhook registration, and no token storage until an operator explicitly applies a reviewed plan.

## Scope

- Issues: title, body, labels, state transitions, assignee changes.
- Pull requests: title, body, review state, branch names, merge/close events.
- Discussions: title, category, accepted answer metadata when available.
- Comments: issue, PR review, PR inline, and discussion comments.

## Required controls

- Convert imported items into scoped observations with workspace, profile, peer/repository, source URL, external ID, checksum, and observed timestamp.
- Rate-limit API calls and surface remaining budget in diagnostics.
- Support bounded backfill by repository, since/until time, item type, and max pages.
- Deduplicate by stable external ID plus checksum.
- Redact secrets before storage and preserve redaction summary.
- Preview import counts and sample records before writing observations.

## Non-goals

- No GitHub writes or bot replies in the first connector.
- No broad organization crawl without explicit repository allowlists.
- No background daemon until local preview/import behavior is tested.

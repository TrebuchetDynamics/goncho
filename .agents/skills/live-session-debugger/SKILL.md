---
name: live-session-debugger
description: Audit Gormes sessions under ~/.gormes. Use when inspecting agent sessions, gateway state, indexes, logs, locks, or fix plans.
---

# Live Session Debugger

Audit local Gormes profile sessions and memory at `/home/xel/.gormes` read-only first, so Goncho work can learn from live agent behavior. Runtime repair is secondary and only done when stale state blocks useful session/memory evidence.

## Quick start

```bash
node .agents/skills/live-session-debugger/scripts/audit-gormes-sessions.mjs --root /home/xel/.gormes
```

## Workflow

1. Default target root to `/home/xel/.gormes` unless the user names another path.
2. Run the deterministic audit script and treat the output as the initial session/memory evidence inventory.
3. Inspect minimal supporting files only when needed:
   - `sessions/index.yaml` and `profiles/*/sessions/index.yaml`
   - `memory/MEMORY.md`, `memory/USER.md`, `workspace/memory/*.md`, and profile-local memory files if present
   - `gateway_state.json`, `profiles/*/gateway_state.json`
   - `tools/audit.jsonl`, `subagents/runs.jsonl`, lifecycle/install logs
   - `gateway.pid` only to distinguish live vs stale profile evidence
4. Group Goncho-improvement opportunities by profile/session coverage, durable-memory shape, stale or conflicting memory signals, tool/subagent outcomes, and only then runtime hygiene.
5. Do not mutate local state until the user approves the plan.

## Entry protocol

- Trivial status/audit request: run the audit script read-only and summarize session/memory evidence for Goncho.
- Medium ambiguity: audit `/home/xel/.gormes`, state assumptions, and ask only if a destructive/restart action is needed.
- High risk: stop before deleting sessions, truncating logs, touching `memory.db*`, editing auth/env files, killing processes, rewriting indexes, or exposing private memory content.

## Topology check

- Which agents/profiles exist and which sessions are indexed?
- Which durable memory files exist, how fresh are they, and do they show Goncho-relevant signal without exposing raw content?
- Are JSON/YAML/JSONL state files parseable enough to trust?
- Are locks, WAL files, logs, caches, and generated mirrors current or stale?
- Could a fix break active Telegram/gateway routing or lose memory/session history?

## Verification gate

Before declaring done, provide:

- audit command and timestamp;
- findings with evidence paths, severity, and confidence;
- safe fix order;
- blockers requiring user approval;
- post-fix validation commands.

## Red lines

- Never print secrets from `.env`, `auth.json`, tokens, chat payloads, raw private memory, or private session content.
- Never delete or rewrite session indexes, databases, WAL/SHM files, memory files, or logs without explicit approval and backup/quiescence steps.
- Never kill gateway/profile processes without confirming the owning PID and restart path.
- Do not turn this into general Gormes ops unless runtime state blocks Goncho-relevant session/memory evidence.
- Treat `~/.gormes` as live local runtime state, not source-controlled project content.

## Output contract

End with:

```text
LIVE_SESSION_AUDIT_VALIDATED: yes|no
LIVE_SESSION_AUDIT_DECISION: clean|plan_ready|blocked|needs_approval
```

## References

- [Audit checklist](references/audit-checklist.md)

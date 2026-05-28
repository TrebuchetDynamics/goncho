# Live Gormes Session and Memory Audit Checklist

## Evidence boundaries

Collect metadata, not transcript content. Prefer counts, mtimes, sizes, parse status, profile names, session counts, memory-file freshness, and topic counts. Redact token-like values and avoid dumping chat/session/memory payloads.

## Categories

### Goncho evidence focus

- Profile session indexes show which agents have live usage evidence.
- Durable memory files show freshness, size, structure counts, and Goncho-term counts without printing raw memory.
- Tool/subagent metadata logs can reveal repeated failure/success patterns useful for Goncho memory, negative-memory, and lifecycle work.
- Runtime health findings matter mainly when they make session/memory evidence stale, incomplete, or unsafe to trust.

### Data safety

- `memory.db`, `sessions.db`, `*.db-wal`, and `*.db-shm` exist and are not obviously orphaned while a writer is active.
- `memory/MEMORY.md`, `memory/USER.md`, workspace mirrors, and profile-local memory files exist if expected.
- Lock files are recent when a writer is active; old locks are suspicious, not automatically removable.

### Session routing

- Root `sessions/index.yaml` and each `profiles/*/sessions/index.yaml` are readable.
- Session IDs are unique enough for the intended profile/source.
- `updated_at` and `metadata_updated_at` are plausible relative to current time.
- Empty indexes may be normal for inactive profiles; classify as info unless a profile is active.

### Process health

- `gateway.pid` and `profiles/*/gateway.pid` contain either a numeric PID or parseable JSON with a `pid` field.
- PID exists and command line looks like the expected Gormes/gateway process.
- Missing PID with recent gateway state/log activity means state may be stale or gateway died.

### State files

- `gateway_state.json`, profile gateway states, `channel_directory_sources.json`, and config JSON/TOML are parseable/readable.
- JSONL logs (`tools/audit.jsonl`, `subagents/runs.jsonl`, lifecycle logs) have no malformed recent records.
- Large logs/caches are hygiene findings unless they block runtime behavior.

### Security and privacy

- Confirm existence and permissions of `.env`/`auth.json` without reading values unless explicitly necessary.
- Flag world-readable secret-bearing files.
- Do not include raw user IDs/chat IDs in final prose unless already provided by the operator or needed for routing diagnosis.
- Do not quote raw memory or session text; summarize only counts, freshness, structure, and Goncho-relevant categories.

## Fix planning order

1. Back up live state before edits: databases, indexes, memory files, gateway state, and logs relevant to the finding.
2. Stop or quiesce writers before touching DB/WAL/SHM/index/memory files.
3. Repair parse errors or stale mirrors before changing routing.
4. If improving Goncho, convert findings into bounded Goncho hypotheses and tests before changing production behavior.
5. Restart gateways only after config/state checks pass.
6. Rerun the audit script and one targeted runtime smoke check.

## Severity guide

- Critical: likely data loss, secret exposure, corrupt DB/index, or wrong live routing.
- High: gateway down/stuck, PID mismatch, active stale lock, repeated failing runs.
- Medium: malformed non-critical JSONL, stale profile state, high log/cache growth.
- Low/info: inactive profiles, empty indexes, old cache files, missing optional artifacts.

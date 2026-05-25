# Security Policy

Goncho is local-first memory infrastructure for agent hosts. Treat it as sensitive local data: memory can include user prompts, tool outputs, project facts, review decisions, snapshots, and derived recall evidence.

## Vulnerability reporting

Please report suspected vulnerabilities privately to the maintainers before public disclosure. Include:

- affected Goncho version or commit;
- package/command involved (`service`, `http`, `cmd/goncho`, `cmd/goncho-server`, connector plan, or docs);
- reproduction steps using local test data where possible;
- whether local files, secrets, non-loopback serving, prompt injection, redaction, or snapshot exports are involved.

Do not include real secrets or private memory exports in a report. Redact tokens and provide a minimal fixture.

## Supported versions

Goncho is pre-1.0. Security fixes target the current main branch and the latest published public module version when practical. Pin production hosts to a reviewed commit or tag and run local smoke tests before upgrading.

Use:

```bash
go run ./cmd/goncho version --json
go run ./cmd/goncho upgrade-check --json --current <pinned> --latest <trusted-release>
```

## Local files and permissions

Goncho stores SQLite databases, optional markdown mirrors, vector indexes, image refs, preferences, and snapshot/export files on local disk. Recommended defaults:

- keep DBs and exports outside world-readable directories;
- use `0700` parent directories and `0600` config/export files where possible;
- treat portable JSONL/Markdown exports and snapshot manifests as sensitive artifacts;
- back up before retention/import/restore operations;
- do not commit generated DBs, exports, or local connector configs unless intentionally sanitized.

## Non-loopback binds

`goncho-server serve` defaults to `127.0.0.1:8765`. Non-loopback binds are shared-server mode and require an explicit auth token guard today; stronger auth/RBAC remains future work.

Do not expose directly to the internet. For local shared service experiments, prefer Docker Compose with host publishing limited to `127.0.0.1` and review `docs/server-mode-threat-model.md` plus `docs/deployment-local-shared-service.md`.

## Prompt injection and quarantine

Prompt, tool, and transcript content can be adversarial. Goncho records evidence; it does not make retrieved memory authoritative. Host agents must verify live state before acting.

Quarantine or review when content attempts to override system/developer instructions, requests secret exfiltration, asks to disable safety checks, or conflicts with trusted evidence. Prompt-injection evidence should remain auditable without being promoted to active conclusions.

## Redaction

Host hook capture redacts common authorization headers, JSON secrets, private key material, and environment-style secrets before storage. Large hook payloads are truncated. Redaction is a defense-in-depth layer, not permission to send secrets to memory.

Connector plans are preview-first. Review generated patches before applying host configuration, and keep `--apply` disabled until a connector has golden tests and host smoke coverage.

## Snapshot exports and restore

Snapshot manifests, portable JSONL, and Markdown exports preserve provenance, lifecycle state, review state, checksums, workspace/profile scope, and stable IDs. They can reveal sensitive project history even when current recall is scoped.

Before sharing or restoring exports:

- preview imports before apply;
- fail closed on stable-ID conflicts;
- preserve redaction summaries and checksums;
- record the operator/admin actor for restore and retention changes;
- verify that archived/stale/private memories are not accidentally exposed to another workspace or profile.

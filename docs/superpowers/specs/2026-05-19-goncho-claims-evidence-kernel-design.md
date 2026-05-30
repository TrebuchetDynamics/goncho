# Goncho Claims/Evidence Kernel Foundation

Date: 2026-05-19

## Background

`docs/opensource-memory-systems/analysis/METAANALYSIS-MEMORY-SYSTEMS.md` frames Goncho as a trust-preserving context architecture, not a vector database or passive note store. The first durable layer should preserve scoped, timestamped evidence before Goncho derives claims, beliefs, orientation packs, graph edges, or negative memories from that evidence.

Agentmemory is useful background for hook-native capture and audit discipline, but this design stays embedded in Goncho: no Node sidecar, iii engine, REST daemon, MCP expansion, or external vector dependency.

## Scope

This slice adds the raw evidence lane for Goncho:

- `RunMigrations(db *sql.DB) error` creates Goncho-owned observation and audit tables.
- `Observe` records redacted, size-bounded lifecycle evidence.
- `ListObservations` reads evidence by exact filters.
- `AuditTrail` reads append-only audit events.
- `Service` gets thin wrappers over those package-level APIs.

The implementation is intentionally narrow. It does not derive durable memories from observations and does not retrofit audit into the current live memory/tool/conclusion stack.

## Decisions

- Use separate `goncho_observations` and `goncho_audit_events` tables.
- Do not reuse `turns`; turns are conversational history, observations are lifecycle evidence.
- Use `SessionKey` / `session_key`, matching the current Goncho service vocabulary.
- Store observations as evidence, not claims. Claim lifecycle states such as `canonical`, `outdated`, `quarantined`, and `review_required` belong to a later claims layer.
- V1 audit covers `Observe` only. Auditing `LocalMarkdownMemoryStore`, service conclusions, session deletion, dynamic agents, and webhooks is follow-up work.
- Existing `DeleteSession` and `DeleteWorkspace` behavior is unchanged. Observation deletion/governance is a later slice with its own audit contract.
- Observations do not feed `Search` or `Context` in v1. Raw evidence is queryable through `ListObservations` only until a claims/promotion layer exists.
- Keep Go APIs small: `Observe`, `ListObservations`, `AuditTrail`, and `RunMigrations`.
- Keep MCP/tool exposure out of this slice.
- Keep file paths, tool names, host hook names, source refs, and multiple-file lists in `Metadata map[string]string`.
- Use package-level APIs as the core implementation and `Service` wrappers for configured workspace defaults.
- Patch the current test harness by replacing forbidden imports of `github.com/TrebuchetDynamics/gormes-agent/internal/transcript` with Goncho-local stable JSON golden helpers.

## Schema

`RunMigrations` applies these PRAGMAs and then runs idempotent DDL in a transaction:

```sql
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA busy_timeout = 2000;
PRAGMA foreign_keys = ON;
```

`goncho_observations`:

```text
id TEXT PRIMARY KEY
kind TEXT NOT NULL CHECK(kind IN ('session_start','user_prompt','tool_call','tool_result','tool_error','assistant_response','compact','session_end','custom'))
workspace_id TEXT NOT NULL DEFAULT ''
peer_id TEXT NOT NULL DEFAULT ''
session_key TEXT NOT NULL DEFAULT ''
context_id TEXT NOT NULL DEFAULT ''
input TEXT NOT NULL DEFAULT ''
output TEXT NOT NULL DEFAULT ''
success INTEGER CHECK(success IN (0, 1) OR success IS NULL)
metadata_json TEXT NOT NULL DEFAULT '{}'
input_truncated INTEGER NOT NULL DEFAULT 0 CHECK(input_truncated IN (0, 1))
output_truncated INTEGER NOT NULL DEFAULT 0 CHECK(output_truncated IN (0, 1))
input_original_bytes INTEGER NOT NULL DEFAULT 0
output_original_bytes INTEGER NOT NULL DEFAULT 0
redacted INTEGER NOT NULL DEFAULT 0 CHECK(redacted IN (0, 1))
redaction_count INTEGER NOT NULL DEFAULT 0
checksum TEXT NOT NULL
observed_at INTEGER NOT NULL
```

`goncho_audit_events`:

```text
id TEXT PRIMARY KEY
action TEXT NOT NULL CHECK(action IN ('observe'))
target_type TEXT NOT NULL CHECK(target_type IN ('observation'))
target_id TEXT NOT NULL
workspace_id TEXT NOT NULL DEFAULT ''
peer_id TEXT NOT NULL DEFAULT ''
session_key TEXT NOT NULL DEFAULT ''
reason TEXT NOT NULL
metadata_json TEXT NOT NULL DEFAULT '{}'
created_at INTEGER NOT NULL
```

Observation indexes use `observed_at`; audit indexes use `created_at`. Add plain time indexes plus targeted exact-filter indexes for workspace, peer, session, context, kind, audit target, and audit action. Do not add FTS, vector, or JSON metadata indexes in v1.

## Public API

Observation kinds are semantic, not host-hook names:

```text
session_start
user_prompt
tool_call
tool_result
tool_error
assistant_response
compact
session_end
custom
```

`custom` requires non-empty `Metadata["custom_kind"]`.

Package-level functions:

```go
func RunMigrations(db *sql.DB) error
func Observe(ctx context.Context, db *sql.DB, p ObservationParams) (ObservationResult, error)
func ListObservations(ctx context.Context, db *sql.DB, q ObservationQuery) (ObservationList, error)
func AuditTrail(ctx context.Context, db *sql.DB, q AuditQuery) (AuditResult, error)
```

Service wrappers:

```go
func (s *Service) Observe(ctx context.Context, p ObservationParams) (ObservationResult, error)
func (s *Service) ListObservations(ctx context.Context, q ObservationQuery) (ObservationList, error)
func (s *Service) AuditTrail(ctx context.Context, q AuditQuery) (AuditResult, error)
```

Service wrappers default only `WorkspaceID` to `s.workspaceID`. They do not default peer, session, or context. For service query wrappers only, `WorkspaceID: "*"` means "do not apply the service workspace default." `Service.Observe` rejects `WorkspaceID: "*"`.

Package-level filters use simple semantics: empty filter means no filter; non-empty filter means exact match. Package-level APIs do not treat `"*"` specially.

Observation APIs use `PeerID`, not `Peer`, because evidence rows need a canonical identifier. Older higher-level service APIs can keep `Peer` for ergonomics.

Typed sentinel errors:

```go
var (
    ErrObservationConflict      = errors.New("goncho: observation conflict")
    ErrObservationNotFound      = errors.New("goncho: observation not found")
    ErrObservationSchemaMissing = errors.New("goncho: observation schema missing")
    ErrObservationInvalid       = errors.New("goncho: invalid observation")
)
```

Use these only where callers need branching. Lower-level failures should wrap with `fmt.Errorf("goncho: ...: %w", err)`.

## Safety

`Observe` stores only sanitized evidence:

- Input/output and metadata values are coerced to valid UTF-8.
- IDs and metadata keys must already be valid UTF-8, non-NUL, and within byte limits.
- Metadata keys are trimmed, must be non-empty, and duplicate-after-trim keys are invalid.
- Metadata values preserve whitespace except redaction.
- Metadata values are redacted before size validation.
- Input/output are redacted before truncation.
- Oversized input/output are truncated at valid UTF-8 boundaries.

Private v1 byte limits:

```text
observation id: 256
workspace/peer/session/context ids: 512
metadata key: 128
metadata value: 4 KiB
metadata JSON: 16 KiB
input: 16 KiB
output: 64 KiB
```

V1 redacts obvious secrets only:

- `<private>...</private>` blocks, case-insensitive.
- Authorization bearer headers.
- `.env` style keys containing `SECRET`, `TOKEN`, `PASSWORD`, `API_KEY`, or `PRIVATE_KEY`.
- JSON fields whose keys contain secret/token/password/api_key/private_key/authorization.
- PEM private key blocks.
- Common API key prefixes such as `sk-`, `ghp_`, and `github_pat_`.

Use stable typed markers such as `[REDACTED:authorization]` and `[REDACTED:env_secret]`. Do not keep partial secret prefixes/suffixes. Do not run a broad entropy/base64 heuristic and do not add prompt-injection quarantine in v1.

## Idempotency

Generated observation IDs use:

```text
obs_<unixnano>_<randhex8>
```

Caller-provided observation IDs are preserved after validation. Observation IDs are globally unique primary keys.

If a caller-provided ID already exists:

- Same canonical checksum: return the stored observation, first matching observe audit ID, and `Replayed: true`.
- Different checksum: return a conflict error and write nothing.
- Missing observe audit event: return an error and do not repair.

The checksum covers the canonical stored payload after redaction/truncation and deterministic metadata encoding. It excludes ID and `ObservedAt`.

## Time

Observations use `ObservedAt` / `observed_at`, stored as Unix nanoseconds. Zero `ObservedAt` defaults to current UTC time. `ObservationQuery.Since` and `Until` are inclusive bounds over `ObservedAt`.

Audit events use `CreatedAt` / `created_at`, stored as Unix nanoseconds at Goncho write time. Observe audit time is not copied from the observation event time.

## Audit

V1 audit action and target types are intentionally narrow:

```text
action = observe
target_type = observation
```

`Observe` inserts the observation and its audit event in the same transaction. Successful `Observe` always returns a non-empty audit ID. Idempotent replay writes no new audit event.

Observe audit metadata contains generated safety summary only:

```text
redacted
redaction_count
redaction_kinds
input_truncated
output_truncated
input_original_bytes
output_original_bytes
```

Observation caller metadata is not copied into observe audit metadata. Audit reason defaults to the action string and is always redacted before storage.

## Query Behavior

`ListObservations` filters by:

- `WorkspaceID`
- `PeerID`
- `SessionKey`
- `ContextID`
- `Kinds`
- `Success`
- inclusive `Since` / `Until`
- `Limit`

`AuditTrail` filters by:

- `Action`
- `TargetType`
- `TargetID`
- `WorkspaceID`
- `PeerID`
- `SessionKey`
- inclusive `Since` / `Until`
- `Limit`

Default limit is 50 and max limit is 500. Results are newest-first. Empty results return non-nil empty slices and `Count: 0`. `Count` is the returned count, not a total available count.

Read APIs fail on corrupt rows such as malformed metadata JSON. They do not silently skip corrupted data.

## Tests

Focused tests should cover:

- `RunMigrations` creates the two tables and is idempotent.
- `Observe` writes an observation and observe audit event transactionally.
- `Observe` redacts secrets before storage.
- `Observe` truncates oversized payloads at valid UTF-8 boundaries and records safety fields.
- `Observe` stores caller-provided `ObservedAt` after UTC normalization, defaults zero `ObservedAt`, and keeps audit `CreatedAt` as write time.
- Caller-provided observation ID replay is idempotent for the same canonical payload.
- Caller-provided observation ID conflict fails for different canonical payload.
- Missing observe audit on replay fails.
- Observation sentinel errors work with `errors.Is`.
- `ListObservations` filters by workspace, peer, session, context, kind, success, and time.
- `AuditTrail` filters by action, target, workspace, peer, session, and time.
- `Service` wrappers default workspace only and support query wildcard `WorkspaceID: "*"`.
- Malformed metadata JSON makes read APIs fail.

Baseline `go test . -count=1` currently fails before this slice because `proof_matrix_test.go` and `recall_benchmark_test.go` import `github.com/TrebuchetDynamics/gormes-agent/internal/transcript`, which is not importable from this module. The preflight harness fix is part of this slice. Slice validation should require targeted observation/audit tests to pass and still run the full suite as a global signal.

## Non-Goals

- No vector embeddings.
- No RRF or retrieval rewrite.
- No observation FTS.
- No boot orientation pack.
- No MCP/tool additions.
- No dashboard or viewer.
- No prompt-injection classifier/quarantine.
- No claim table or claim lifecycle states.
- No automatic memory derivation/compression.
- No audit retrofit for current memory/tool/conclusion/session/dynamic-agent/webhook mutations.
- No generic scope table.
- No subject/entity/file-path columns.
- No exported redactor API.
- No commits as part of this slice.

## Follow-Up Slices

1. Claims layer: distilled facts, procedures, negative memories, confidence, validity intervals, review state, and evidence links.
2. Orientation pack: primer, latest handoff, canonical facts, recent episodes, warnings, dead ends, unresolved conflicts, and citations.
3. Mutation audit: `LocalMarkdownMemoryStore` store/update/forget, service conclusions, session deletion, dynamic agents, and webhooks.
4. Evidence search: optional observation FTS and file-history indexes.
5. Small tool surface: context, search, remember, review, handoff.
6. Stewardship: review queues, duplicate/conflict detection, verification, and promotion.

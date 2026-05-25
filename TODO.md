# Goncho TODO — Productized Runtime UX Gap Roadmap

This TODO tracks the highest-value gaps between Goncho and the source-pinned `agentmemory` reference, plus selected `mem0` product/API lessons. The gap is mostly **productized integration/runtime UX**, not core memory theory.

- Agentmemory reference repo: `docs/opensource-memory-systems/agentmemory`
- Mem0 comparison/probe references: `scripts/bench_mem0_locomo.py`, `docs/benchmarks/external-backend-adapters.md`, `docs/benchmarks/locomo-backend-comparison.md`
- Source-pinned Goncho mirror: `memorymirror.ArchitectureManifest()` in `memorymirror/architecture.go`
- Backlog mirror: `memorymirror/ImplementationBacklog()` in `memorymirror/backlog.go`
- Goncho principle: keep core memory trust-preserving and local-first; put broad runtime UX behind explicit adapters, tools, and server mode.

Priority scale:

- **P0** = most important, blocks Goncho feeling like a product.
- **P1** = high-leverage product surface after P0 exists.
- **P2** = major capability expansion with clear adapter seams.
- **P3** = advanced/team/runtime coordination.
- **P4** = media/connector breadth.
- **P5** = operational polish and ecosystem packaging.
- **P6** = optional parity/experiments; do not compromise Goncho's trust model for these.

---

## P0 — Turnkey Goncho Server + Health + Minimal Runtime Product

### Why this matters

Agentmemory feels usable immediately because a user can run a server and see a working memory product:

- `npx @agentmemory/agentmemory`
- HTTP server and MCP server
- health endpoint
- demo/seed path
- viewer link
- `doctor`/diagnostics paths

Goncho is currently excellent as an embedded Go SDK, but users who are not already inside Gormes do not get a polished standalone runtime.

### Agentmemory references

- `docs/opensource-memory-systems/agentmemory/src/cli.ts`
- `docs/opensource-memory-systems/agentmemory/src/mcp/server.ts`
- `docs/opensource-memory-systems/agentmemory/src/mcp/standalone.ts`
- `docs/opensource-memory-systems/agentmemory/src/mcp/rest-proxy.ts`
- `docs/opensource-memory-systems/agentmemory/src/health/monitor.ts`
- `docs/opensource-memory-systems/agentmemory/src/health/thresholds.ts`
- `docs/opensource-memory-systems/agentmemory/dist/standalone.mjs`
- `docs/opensource-memory-systems/agentmemory/dist/docker-compose.yml`
- `docs/opensource-memory-systems/agentmemory/.env.example`

### Goncho seams today

- `service.NewService`
- `memory.OpenSqlite`
- `service.RunMigrations`
- `http/` local adapter
- `NewGonchoContextTool`, `NewGonchoSearchTool`, `NewGonchoRecallTool`, `NewGonchoRememberTool`, `NewReviewTool`, `NewGonchoHandoffTool`
- `goncho-adapter-api.md`

### Deliverables

- [x] Add `cmd/goncho-server` as a first-class local memory server.
- [x] Add `goncho-server init` to create a local config and SQLite DB path.
- [x] Add `goncho-server serve` with HTTP API and MCP-compatible tool transport.
- [x] Add `goncho-server health` returning JSON health, version, DB status, migration status, and tool availability.
- [x] Add `goncho-server demo` that seeds a tiny project-memory scenario and proves recall/context.
- [x] Add `goncho-server doctor` that checks DB path, migrations, write permissions, port conflicts, and public tool registration.
- [x] Add `make server-smoke` that starts the server on a random local port, calls health, writes one memory, recalls it, and shuts down cleanly.

### Acceptance tests

- [x] `go test ./cmd/goncho-server ./http ./service`
- [x] `make server-smoke`
- [x] `go test ./...`
- [x] `git diff --check`

### Non-goals

- Do not make the server mandatory for Gormes or embedded users.
- Do not add cloud dependencies.
- Do not auto-bind to public interfaces without explicit authentication.

---

## P1 — Automatic Hook Installation and Agent Connectors

### Why this matters

Agentmemory's strongest UX is zero-manual capture. Users install/connect once and hooks capture prompts, tool use, failures, compaction, subagents, and session lifecycle.

Goncho has `svc.CaptureHostHook`, but hosts still need to forward events manually. That is architecturally clean, but not yet productized.

### Agentmemory references

- `docs/opensource-memory-systems/agentmemory/plugin/hooks/hooks.json`
- `docs/opensource-memory-systems/agentmemory/plugin/hooks/hooks.codex.json`
- `docs/opensource-memory-systems/agentmemory/plugin/scripts/session-start.mjs`
- `docs/opensource-memory-systems/agentmemory/plugin/scripts/prompt-submit.mjs`
- `docs/opensource-memory-systems/agentmemory/plugin/scripts/pre-tool-use.mjs`
- `docs/opensource-memory-systems/agentmemory/plugin/scripts/post-tool-use.mjs`
- `docs/opensource-memory-systems/agentmemory/plugin/scripts/post-tool-failure.mjs`
- `docs/opensource-memory-systems/agentmemory/plugin/scripts/pre-compact.mjs`
- `docs/opensource-memory-systems/agentmemory/plugin/scripts/stop.mjs`
- `docs/opensource-memory-systems/agentmemory/plugin/scripts/session-end.mjs`
- `docs/opensource-memory-systems/agentmemory/plugin/.claude-plugin/plugin.json`
- `docs/opensource-memory-systems/agentmemory/plugin/.codex-plugin/plugin.json`
- `docs/opensource-memory-systems/agentmemory/src/cli/connect/claude-code.ts`
- `docs/opensource-memory-systems/agentmemory/src/cli/connect/codex.ts`
- `docs/opensource-memory-systems/agentmemory/src/cli/connect/cursor.ts`
- `docs/opensource-memory-systems/agentmemory/src/cli/connect/gemini-cli.ts`
- `docs/opensource-memory-systems/agentmemory/src/cli/connect/hermes.ts`
- `docs/opensource-memory-systems/agentmemory/src/cli/connect/openclaw.ts`
- `docs/opensource-memory-systems/agentmemory/src/cli/connect/pi.ts`

### Goncho seams today

- `service.CaptureHostHook`
- `HostHookEvent`
- `service.Observe`
- `service.CreateMessages`
- `service.SessionSummary`
- `plugins` write queue
- Gormes adapter contract in `goncho-adapter-api.md`

### Deliverables

- [ ] Add `goncho connect claude-code --dry-run` that prints hook files/config changes without mutating.
- [ ] Add `goncho connect codex --dry-run` for Codex hook/MCP wiring.
- [ ] Add `goncho connect cursor --dry-run` for MCP config wiring.
- [ ] Add `goncho connect gemini-cli --dry-run` for MCP config wiring.
- [ ] Add `goncho connect hermes --dry-run` with explicit handoff to `gormes-agent`/Hermes adapter when applicable.
- [ ] Add `goncho connect gormes` as the canonical first-party integration path.
- [x] Add host event schemas for prompt, assistant response, pre-tool, post-tool, tool failure, compaction, subagent start/stop, stop, and session end.
- [x] Add privacy/redaction filters before events reach `CaptureHostHook`.
- [x] Add docs showing which hook events are captured, which are ignored, and which require host-specific permission.

### Acceptance tests

- [ ] Golden-file tests for generated hook configs.
- [ ] Tests that every generated hook maps to a `HostHookEvent` type.
- [x] Tests that secrets and large tool payloads are redacted/truncated before storage.
- [ ] `go test ./...`

### Non-goals

- Do not silently mutate user agent configs without `--apply`.
- Do not install global hooks by default.
- Do not claim full agentmemory connector parity until each connector has a smoke test.

---

## P2 — Real-time Viewer, Session Replay, and Transcript Import

### Why this matters

Agentmemory's viewer makes memory legible. Users can see sessions, replay timelines, inspect memory growth, and debug why retrieval worked or failed. Goncho has audit APIs, recall traces, and benchmark artifacts, but no first-class visual product surface.

### Agentmemory references

- `docs/opensource-memory-systems/agentmemory/src/viewer/server.ts`
- `docs/opensource-memory-systems/agentmemory/src/viewer/index.html`
- `docs/opensource-memory-systems/agentmemory/dist/viewer/index.html`
- `docs/opensource-memory-systems/agentmemory/src/replay/jsonl-parser.ts`
- `docs/opensource-memory-systems/agentmemory/src/replay/timeline.ts`
- `docs/opensource-memory-systems/agentmemory/src/functions/replay.ts`
- `docs/opensource-memory-systems/agentmemory/assets/demo.gif`
- `docs/opensource-memory-systems/agentmemory/assets/iii-console/states.png`
- `docs/opensource-memory-systems/agentmemory/assets/iii-console/traces-waterfall.png`
- `docs/opensource-memory-systems/agentmemory/assets/iii-console/workers.png`

### Goncho seams today

- `service.ListObservations`
- `service.AuditTrail`
- `service.Recall` trace
- `service.Context` representation
- session summaries
- review queue APIs
- `docs/benchmarks/results/*` JSON artifacts

### Deliverables

- [ ] Add read-only `goncho-viewer` or `goncho-server --viewer` endpoint.
- [ ] Show health, DB path, workspace/profile/session counts, latest observations, latest conclusions, and review queue status.
- [ ] Add recall trace viewer: candidates, selected/rejected, provenance, warnings, query expansion, vector/lexical/graph signals.
- [ ] Add orientation-pack viewer: what entered context and why.
- [ ] Add session timeline view from observations/messages/summaries.
- [ ] Add Claude JSONL transcript import preview.
- [ ] Add transcript import apply path with deduplication and provenance.
- [ ] Add redaction view: show what was dropped or truncated, without exposing secrets.

### Acceptance tests

- [ ] JSON API tests for viewer endpoints.
- [ ] Snapshot tests for replay timeline output from a tiny Claude JSONL fixture.
- [ ] Import tests proving idempotent transcript import.
- [ ] Browser asset build smoke if a frontend bundle is added.

### Non-goals

- No cloud dashboard.
- No write operations from viewer until auth and audit are explicit.

---

## P3 — MCP Tool Catalog Expansion Without Losing the Small Core Surface

### Why this matters

Agentmemory exposes 50+ MCP tools. Goncho intentionally exposes a smaller trusted surface, which is a strength. But users coming from agentmemory expect familiar high-level operations: sessions, file history, timeline, audit, slots, snapshots, lessons, facets, diagnostics, and graph queries.

The right Goncho move is not to copy every tool into core. It is to expose a layered tool catalog where broad aliases call small trusted APIs.

### Agentmemory references

- `docs/opensource-memory-systems/agentmemory/src/mcp/tools-registry.ts`
- `docs/opensource-memory-systems/agentmemory/dist/tools-registry-BFwOoyLn.mjs`
- `docs/opensource-memory-systems/agentmemory/src/functions/actions.ts`
- `docs/opensource-memory-systems/agentmemory/src/functions/audit.ts`
- `docs/opensource-memory-systems/agentmemory/src/functions/checkpoints.ts`
- `docs/opensource-memory-systems/agentmemory/src/functions/facets.ts`
- `docs/opensource-memory-systems/agentmemory/src/functions/frontier.ts`
- `docs/opensource-memory-systems/agentmemory/src/functions/graph-retrieval.ts`
- `docs/opensource-memory-systems/agentmemory/src/functions/lessons.ts`
- `docs/opensource-memory-systems/agentmemory/src/functions/slots.ts`
- `docs/opensource-memory-systems/agentmemory/src/functions/snapshot.ts`
- `docs/opensource-memory-systems/agentmemory/src/functions/timeline.ts`
- `docs/opensource-memory-systems/agentmemory/src/functions/verify.ts`

### Goncho seams today

- `memorymirror.NewToolRegistry`
- `memorymirror.ArchitectureManifest()`
- public Goncho tools: context/search/recall/remember/review/handoff
- memory slots API
- action graph API
- snapshots API
- audit/review APIs
- graph recall provenance

### Deliverables

- [ ] Add a documented `goncho/tools/compat` catalog for agentmemory-style aliases.
- [ ] Promote existing memorymirror aliases (`memory_save`, `memory_smart_search`, `memory_recall`, `memory_profile`) into documented compatibility tools if stable.
- [ ] Add `memory_timeline` backed by observations/timeline annotations.
- [ ] Add `memory_audit` backed by `AuditTrail`.
- [ ] Add `memory_slot_*` backed by Goncho memory slot APIs.
- [ ] Add `memory_snapshot_*` backed by deterministic snapshot manifests only; git operations stay adapter-owned.
- [ ] Add `memory_graph_query` backed by recall graph provenance.
- [ ] Add `memory_verify` backed by recall provenance plus live-check warnings.
- [ ] Add `memory_diagnose` backed by diagnostics/queue status.
- [ ] Mark every compatibility tool as delivered, partial, adapter-owned, deferred, or excluded in `memorymirror.ArchitectureManifest()`.

### Acceptance tests

- [ ] Tool registry manifest tests verify every documented tool exists in the manifest.
- [ ] Tool execution tests prove each delivered compatibility tool calls public service APIs only.
- [ ] No compatibility tool writes directly to SQLite.
- [ ] `go test ./memorymirror ./service ./toolmeta ./...`

### Non-goals

- Do not expose a giant unreviewed mutating catalog by default.
- Do not hide dangerous operations behind friendly names.

---

## P4 — Local Dense Embeddings and Vision Search

### Why this matters

Agentmemory has local embeddings and vision/image search. Goncho has an optional vector seam and image metadata storage, but not a bundled local embedding runtime or image embedding/search flow.

This is a credibility gap for users who expect semantic retrieval to work out of the box.

### Agentmemory references

- `docs/opensource-memory-systems/agentmemory/src/providers/embedding/local.ts`
- `docs/opensource-memory-systems/agentmemory/src/providers/embedding/clip.ts`
- `docs/opensource-memory-systems/agentmemory/src/providers/embedding/openai.ts`
- `docs/opensource-memory-systems/agentmemory/src/providers/embedding/voyage.ts`
- `docs/opensource-memory-systems/agentmemory/src/state/vector-index.ts`
- `docs/opensource-memory-systems/agentmemory/src/state/hybrid-search.ts`
- `docs/opensource-memory-systems/agentmemory/src/functions/vision-search.ts`
- `docs/opensource-memory-systems/agentmemory/src/functions/image-refs.ts`
- `docs/opensource-memory-systems/agentmemory/src/utils/image-store.ts`
- `docs/opensource-memory-systems/agentmemory/benchmark/REAL-EMBEDDINGS.md`

### Goncho seams today

- `Config.VectorStore`
- `VectorStore`, `VectorSearchQuery`, `VectorSearchHit`
- RRF fusion with semantic provenance
- `StoreImageMemory`
- `SearchImageMemories`
- image refs/checksums/alt text/metadata with deferred embeddings

### Deliverables

- [ ] Add a local embedding provider interface separate from `VectorStore` storage.
- [ ] Add a simple file-backed local vector index implementation suitable for small projects.
- [ ] Add deterministic fake embedding tests and optional real embedding integration tests gated by build tags/env.
- [ ] Add import/reindex command for existing textual memories.
- [ ] Add vector index diagnostics: dimensions, count, checksum, stale rows, last indexed time.
- [ ] Add image embedding provider interface.
- [ ] Add image search over stored refs/checksums/alt text and optional embeddings.
- [ ] Document privacy and model download implications.

### Acceptance tests

- [ ] `go test ./service -run Vector`
- [ ] `go test ./service -run Image`
- [ ] `make bench-longmemeval-s-smoke` shows no regression.
- [ ] Optional tagged real-embedding smoke documents exact model/version.

### Non-goals

- Do not require Python.
- Do not require hosted embedding APIs.
- Do not make vector search authoritative over evidence/provenance.

---

## P5 — Multi-agent Coordination, Server Mode, and Operational Packaging

### Why this matters

Agentmemory includes multi-agent concepts: leases, signals, mesh/team flows, shared service deployment, and operational packaging. Goncho has local action graph signals and ACL/policy pieces, but no distributed coordination product.

This should stay behind explicit server/team mode because distributed coordination changes the trust and governance model.

### Agentmemory references

- `docs/opensource-memory-systems/agentmemory/src/functions/leases.ts`
- `docs/opensource-memory-systems/agentmemory/src/functions/signals.ts`
- `docs/opensource-memory-systems/agentmemory/src/functions/mesh.ts`
- `docs/opensource-memory-systems/agentmemory/src/functions/team.ts`
- `docs/opensource-memory-systems/agentmemory/src/functions/actions.ts`
- `docs/opensource-memory-systems/agentmemory/src/functions/frontier.ts`
- `docs/opensource-memory-systems/agentmemory/deploy/README.md`
- `docs/opensource-memory-systems/agentmemory/deploy/coolify/README.md`
- `docs/opensource-memory-systems/agentmemory/deploy/fly/README.md`
- `docs/opensource-memory-systems/agentmemory/deploy/railway/README.md`
- `docs/opensource-memory-systems/agentmemory/deploy/render/README.md`
- `docs/opensource-memory-systems/agentmemory/docker-compose.yml`

### Goncho seams today

- local action graph: `UpsertAction`, `ReadActionGraph`, `CompleteAction`, `SignalAction`
- policy ACLs
- workspace/profile scopes
- snapshots/export manifests
- local HTTP adapter

### Deliverables

- [ ] Define `server mode` threat model: auth, profiles, workspaces, audit, backup, retention, and admin operations.
- [ ] Add Postgres adapter plan for team/shared deployments.
- [ ] Add distributed action leases with TTL, owner, renewal, expiration, and audit trail.
- [ ] Add inter-agent signals with read receipts and workspace/profile authorization.
- [ ] Add team feed API with pagination and ACL enforcement.
- [ ] Add Docker image and `docker-compose.yml` for local shared service smoke.
- [ ] Add deployment docs for one conservative target first, not all platforms at once.
- [ ] Add backup/export/restore docs using snapshot manifests.

### Acceptance tests

- [ ] Concurrency tests for leases and expiration.
- [ ] ACL tests for cross-profile/team feed reads.
- [ ] Docker compose smoke starts server, runs health, writes/reads memory, shuts down.
- [ ] No unauthenticated non-loopback bind by default.

### Non-goals

- Do not add P2P mesh sync until server mode is secure and boring.
- Do not weaken local-first embedded mode.

---

## P6 — Ecosystem Polish, Connector Breadth, and Optional Parity Features

### Why this matters

Agentmemory is broad and polished: npm packaging, marketplace metadata, connect commands, doctor/upgrade, examples, security advisories, deployment guides, and many agent-specific docs. Goncho needs enough ecosystem polish to feel excellent without becoming a clone.

### Agentmemory references

- `docs/opensource-memory-systems/agentmemory/package.json`
- `docs/opensource-memory-systems/agentmemory/.github/workflows/ci.yml`
- `docs/opensource-memory-systems/agentmemory/.github/workflows/publish.yml`
- `docs/opensource-memory-systems/agentmemory/.github/security-advisories/*`
- `docs/opensource-memory-systems/agentmemory/examples/python/README.md`
- `docs/opensource-memory-systems/agentmemory/integrations/filesystem-watcher/README.md`
- `docs/opensource-memory-systems/agentmemory/ROADMAP.md` (GitHub connector, Slack/Discord connector, OpenCode hook bus, OpenSSF, SSO/RBAC/audit export)
- `docs/opensource-memory-systems/agentmemory/test/fs-watcher.test.ts`
- `docs/opensource-memory-systems/agentmemory/test/export-import.test.ts`
- `docs/opensource-memory-systems/agentmemory/test/retention.test.ts`
- `docs/opensource-memory-systems/agentmemory/test/retention-access.test.ts`
- `docs/opensource-memory-systems/agentmemory/test/schema-fingerprint.test.ts`
- `docs/opensource-memory-systems/agentmemory/src/cli/onboarding.ts`
- `docs/opensource-memory-systems/agentmemory/src/cli/remove-plan.ts`
- `docs/opensource-memory-systems/agentmemory/src/cli/preferences.ts`
- `docs/opensource-memory-systems/agentmemory/src/providers/circuit-breaker.ts`
- `docs/opensource-memory-systems/agentmemory/src/providers/fallback-chain.ts`
- `docs/opensource-memory-systems/agentmemory/src/functions/disk-size-manager.ts`
- `docs/opensource-memory-systems/agentmemory/src/functions/branch-aware.ts`
- `docs/opensource-memory-systems/agentmemory/src/functions/obsidian-export.ts`
- `docs/opensource-memory-systems/agentmemory/src/eval/self-correct.ts`
- `docs/opensource-memory-systems/agentmemory/src/eval/quality.ts`
- `docs/opensource-memory-systems/agentmemory/test/circuit-breaker.test.ts`
- `docs/opensource-memory-systems/agentmemory/test/obsidian-export.test.ts`
- `docs/opensource-memory-systems/agentmemory/test/working-memory.test.ts`
- `docs/opensource-memory-systems/agentmemory/test/sliding-window.test.ts`
- `docs/opensource-memory-systems/agentmemory/integrations/hermes/README.md`
- `docs/opensource-memory-systems/agentmemory/integrations/openclaw/README.md`
- `docs/opensource-memory-systems/agentmemory/integrations/pi/README.md`
- `docs/opensource-memory-systems/agentmemory/benchmark/COMPARISON.md`
- `docs/opensource-memory-systems/agentmemory/benchmark/LONGMEMEVAL.md`
- `docs/opensource-memory-systems/agentmemory/benchmark/QUALITY.md`
- `docs/opensource-memory-systems/agentmemory/benchmark/SCALE.md`

### Deliverables

- [ ] Add `goncho doctor` for local environment and DB/migration checks.
- [ ] Add `goncho version --json` with module version, git commit if available, DB schema version, and public tool count.
- [ ] Add `goncho upgrade-check` that reports available releases without mutating anything.
- [ ] Add `examples/go/` with minimal service, hook capture, recall trace, memory slots, and viewer/server examples.
- [ ] Add `examples/python/` only if a stable HTTP/server API exists.
- [ ] Add security docs for local files, non-loopback binds, prompt injection quarantine, redaction, and snapshot exports.
- [ ] Add connector docs for Gormes, Hermes, Pi, Cursor, Codex, Claude Code, OpenCode, and generic MCP.
- [ ] Add filesystem watcher connector that imports changed project docs/code as observations behind explicit include/exclude rules.
- [ ] Add GitHub connector plan for issues, PRs, discussions, and comments as scoped observations with rate-limit/backfill controls.
- [ ] Add Slack/Discord connector plan for team chats only after server-mode ACLs and retention are explicit.
- [ ] Add schema-fingerprint command/test so server and adapters can detect incompatible DB/tool schema drift before writes.
- [ ] Add comparison docs that explain Goncho vs mem0 vs agentmemory without star-count hype or benchmark overclaims.
- [ ] Add release checklist linking `make release-smoke`, `make stable-e2e-bench-smoke`, public module smoke, GitHub release, and pkg.go.dev verification.

### Acceptance tests

- [ ] Docs link checker or markdown guard tests for every connector page.
- [ ] Example compile tests for Go examples.
- [ ] Server/API examples gated until P0 lands.
- [ ] Release checklist test verifies commands and current version markers.

### Non-goals

- Do not spend time on broad connector docs before P0/P1 are usable.
- Do not create install instructions for unsupported integrations.

---

## Backlog — Additional Things Worth Stealing From Agentmemory + Mem0

These are not all first-slice priorities. They are candidate product ideas to keep visible while preserving Goncho's small trusted core.

### A. Mem0-style tiny memory API, without losing evidence

Mem0's strongest product lesson is a very small API shape: add, search, update, delete/history with user/session/agent metadata filters. Goncho's service API is richer and safer, but new adopters should get a mem0-simple path.

Deliverables:

- [ ] Add a documented `memory.Add/Search/Update/Delete/History` facade or HTTP aliases over Goncho APIs.
- [ ] Support explicit `user_id`, `agent_id`, `run_id/session_key`, `workspace_id`, `profile_id`, and metadata filters in that facade.
- [ ] Preserve stable caller-supplied IDs for benchmark/import compatibility; never fall back to content-only matching.
- [ ] Add history/audit reads for every memory update/delete, with provenance back to evidence.
- [ ] Add Go examples that feel as short as mem0 quick starts while still showing verification/provenance warnings.

Acceptance tests:

- [ ] Mem0-style facade tests for add/search/update/delete/history.
- [ ] Duplicate-content stable-ID test using LOCOMO-style collisions.
- [ ] Audit/provenance test proving update/delete never erases evidence.

Non-goals:

- Do not copy mem0's hosted/cloud assumptions.
- Do not hide Goncho review state behind a deceptively simple success response.

### B. Conversation-to-memory extraction proposals

Agentmemory and mem0 both make memory feel automatic by extracting durable facts/preferences from conversations. Goncho should do this as proposals, not silent truth writes.

Deliverables:

- [ ] Add `ExtractMemoryProposals` over a bounded session window.
- [ ] Classify proposed operations as `add`, `update`, `supersede`, `delete`, or `noop` with evidence IDs.
- [ ] Route low-confidence, contradictory, or privacy-sensitive proposals into review instead of writing active memory.
- [ ] Add preference extraction for stable user/project preferences with scope and expiry hints.
- [ ] Add procedural/lesson extraction for reusable workflows and known failure patterns.

Acceptance tests:

- [ ] Golden fixtures for add/update/delete/noop proposal classification.
- [ ] Conflict fixture proves contradictory claims enter review.
- [ ] Preference fixture proves profile-scoped storage and no cross-profile leak.

Non-goals:

- Do not run extraction on every hook synchronously.
- Do not promote LLM-extracted claims without evidence and confidence metadata.

### C. Memory stewardship jobs as first-class product UX

Agentmemory has tests and tools around retention, access logs, consistency, cascading updates, confidence, lessons, routines, sentinels, sketches, and facets. Goncho has pieces of lifecycle/trust, but needs operator-facing stewardship.

Deliverables:

- [ ] Add retention/access reports: least-used, stale, high-risk, oversized, and unreviewed memories.
- [ ] Add consistency scan that groups duplicate/conflicting claims by entity/scope/time.
- [ ] Add cascade preview when a canonical claim is superseded and dependent memories/context packs may change.
- [ ] Add `lessons` and `routines` views for reusable engineering workflows, backed by evidence.
- [ ] Add `sentinels` for important facts that should warn if contradicted or missing from context.
- [ ] Add `facets` as lightweight entity/profile/project slices for viewer and context filters.

Acceptance tests:

- [ ] Report tests with deterministic stale/duplicate/conflict fixtures.
- [ ] Cascade preview test with no writes until operator apply.
- [ ] Context test proving sentinels warn without stuffing every sentinel into prompts.

Non-goals:

- Do not add autonomous deletion before retention policy and audit export exist.
- Do not let stewardship jobs mutate trusted memories without review or explicit policy.

### D. MCP compliance and resources/prompts polish

Agentmemory exposes MCP prompts/resources and robust standalone transport behavior. Goncho's `goncho-server` now has a minimal JSON-RPC-compatible `/mcp`; it needs compliance hardening before connector claims broaden.

Deliverables:

- [ ] Add stdio MCP mode for hosts that do not use HTTP.
- [ ] Add SSE/streaming transport only if a target host requires it.
- [ ] Add MCP resources for health, latest observations, recall prompt, profile/context status, and graph stats.
- [ ] Add MCP prompts for evidence-first recall, session handoff, review resolution, and verification-before-action.
- [ ] Add protocol compliance tests for initialize, capabilities, JSON-RPC errors, cancellation/timeouts, and schema shapes.

Acceptance tests:

- [ ] MCP inspector-compatible smoke or protocol fixture test.
- [ ] Tool/resource/prompt manifest tests.
- [ ] Backward-compatible `/mcp` HTTP smoke remains green.

Non-goals:

- Do not expose new mutating tools through MCP until trust class, audit kind, and prompt-safety policy are explicit.

### E. Governance, supply chain, and operational trust

Agentmemory's roadmap includes governance docs, OpenSSF Scorecard, SSO, RBAC, audit export, LTS, and security audit. Goncho should adapt the boring operational parts when server/team mode becomes real.

Deliverables:

- [ ] Add `SECURITY.md`, vulnerability reporting, supported versions, and local-data threat model.
- [ ] Add `GOVERNANCE.md`, `CONTRIBUTING.md`, and maintainer/release decision docs when outside contributors arrive.
- [ ] Add audit export to JSONL/stdout first; defer S3/Loki until server mode needs it.
- [ ] Add RBAC role vocabulary for server mode before implementing enforcement.
- [ ] Add OpenSSF Scorecard/Dependency Review once CI is stable enough.
- [ ] Add semver/API stability policy for `service`, `http`, `cmd/goncho-server`, and integration packages.

Acceptance tests:

- [ ] Docs guard tests for security/contact/supported-version markers.
- [ ] Audit export smoke with redaction preserved.
- [ ] API compatibility checklist in release smoke.

Non-goals:

- Do not perform security theater before public server mode has a concrete threat model.
- Do not promise LTS before v1 API boundaries are frozen.

### F. Onboarding, uninstall, and operator preference UX

Agentmemory invests in first-run onboarding, preferences, connect/remove plans, and doctor autofix guidance. Goncho should make local-first setup understandable without hiding what it will mutate.

Deliverables:

- [ ] Add `goncho-server onboarding` or first-run guidance that explains DB path, config path, loopback bind, MCP URL, and next commands.
- [ ] Add `goncho connect <host> --plan` and `goncho remove <host> --plan` outputs that are symmetric and reversible.
- [ ] Add `goncho preferences` for local operator defaults: DB path, profile/workspace, redaction policy, connector permission level, and default bind address.
- [ ] Add doctor autofix suggestions as patches/commands, not automatic mutation.
- [ ] Add terminal-friendly copy-paste snippets for MCP configs and hook scripts.

Acceptance tests:

- [ ] Golden tests for onboarding text and connect/remove plans.
- [ ] Preference read/write test using temp config paths.
- [ ] Doctor suggestion test proving no mutation without `--apply`.

Non-goals:

- Do not make interactive prompts mandatory for CI or headless agents.
- Do not auto-edit user config files from onboarding.

### G. Provider resilience and background worker safety

Agentmemory has circuit breakers, fallback chains, fetch timeouts, and resilient provider wrappers. Goncho should adapt the pattern for optional LLM/embedding/extraction providers so core memory stays reliable when adapters fail.

Deliverables:

- [ ] Add a provider health model for optional extraction, embedding, reranking, and summarization adapters.
- [ ] Add circuit-breaker state and diagnostics: open/half-open/closed, last error, retry-after, failure counts.
- [ ] Add fallback-chain support where local lexical/graph retrieval remains available when semantic providers fail.
- [ ] Add per-provider timeout and max-payload controls.
- [ ] Surface provider degradation in health, doctor, viewer, and recall warnings.

Acceptance tests:

- [ ] Circuit-breaker tests for repeated failure, cooldown, half-open success, and fail-closed writes.
- [ ] Fallback-chain tests proving lexical/provenance recall still works when embeddings fail.
- [ ] Health test showing degraded optional provider without failing core SQLite service.

Non-goals:

- Do not make hosted providers required.
- Do not let provider fallback change benchmark scoring semantics silently.

### H. Disk budgets, retention, and safe eviction

Agentmemory has disk-size management, image quota cleanup, eviction, auto-forget, and retention tests. Goncho needs bounded local storage behavior before long-lived server mode.

Deliverables:

- [ ] Add DB/image/vector disk usage diagnostics to `goncho-server health` and `doctor`.
- [ ] Add retention policy config: keep forever, max age, max DB size, max image/vector size, per-workspace limits.
- [ ] Add eviction preview that lists candidates and reasons before any deletion/tombstone.
- [ ] Add safe tombstone/archive path preserving audit and stable IDs.
- [ ] Add image/vector refcount cleanup after retention applies.

Acceptance tests:

- [ ] Retention preview fixture with no writes.
- [ ] Eviction apply fixture proves tombstones/audit remain and recall excludes evicted active content.
- [ ] Disk quota smoke over temp DB/image/vector dirs.

Non-goals:

- Do not hard-delete user memory by default.
- Do not evict evidence required by active claims unless policy explicitly allows archival with audit.

### I. Branch-aware and project-state memory

Agentmemory includes branch-aware behavior, post-commit hooks, file indexes, checkpoints, and snapshots. Goncho already has snapshots and stale-code verification; project memory should become branch/worktree aware without doing git operations inside core.

Deliverables:

- [ ] Add optional branch/worktree metadata to observations, conclusions, snapshots, and recall queries.
- [ ] Add post-commit capture adapter plan that records commit hash, changed files, and summary as evidence.
- [ ] Add checkpoint view tying memory snapshots to code snapshots without running git from core service APIs.
- [ ] Add file-index observation import for code/docs with include/exclude and checksum provenance.
- [ ] Add stale-branch warnings when recalling claims captured on a different branch or old commit.

Acceptance tests:

- [ ] Branch metadata isolation/routing tests.
- [ ] Snapshot/checkpoint manifest tests with fake commit IDs.
- [ ] Stale-branch warning test in context/recall output.

Non-goals:

- Do not let Goncho mutate git state.
- Do not treat branch metadata as an authorization boundary; it is retrieval evidence.

### J. Portable export formats and human-editable mirrors

Agentmemory supports export/import and Obsidian export. Goncho has snapshot manifests and local markdown memory; it should make memory portable and inspectable across tools.

Deliverables:

- [ ] Add full JSONL export/import for observations, messages, conclusions, review items, snapshots, and memory slots.
- [ ] Add Obsidian/Markdown export with backlinks, provenance blocks, review status, and stale warnings.
- [ ] Add import preview with counts, conflicts, schema version, redaction summary, and stable-ID collision handling.
- [ ] Add selective export by workspace/profile/session/time range and redaction policy.
- [ ] Add signed/checksummed export manifest for reproducibility.

Acceptance tests:

- [ ] Export/import round-trip preserves IDs, provenance, review state, and tombstones.
- [ ] Obsidian export snapshot test with deterministic markdown.
- [ ] Import preview collision test fails closed without `--apply`.

Non-goals:

- Do not make exported markdown the source of truth.
- Do not export secrets that were redacted/quarantined unless an explicit raw-evidence export mode is designed.

### K. Evaluation, self-correction, and regression gates

Agentmemory carries eval metrics, quality validators, self-correct loops, benchmark-in-CI roadmap, and many retrieval tests. Goncho already has strong benchmark harnesses; next step is productizing eval feedback into development and runtime diagnostics.

Deliverables:

- [ ] Add eval registry that records recall/context failures from benchmark runs as structured improvement candidates.
- [ ] Add self-correction proposals for retrieval misses: query expansion hint, graph edge candidate, extraction gap, stale/contradictory memory, or scope bug.
- [ ] Add benchmark trend reports comparing current branch to frozen baselines.
- [ ] Add runtime feedback labels (`useful`, `wrong`, `stale`, `unsafe`, `missing`) that feed review/negative memory without direct promotion.
- [ ] Add release gate that fails on regression beyond a configured tolerance for smoke datasets.

Acceptance tests:

- [ ] Eval registry test converts a known miss into a structured candidate.
- [ ] Feedback label test writes review/negative-memory evidence without altering active claims.
- [ ] Regression gate test rejects a synthetic metric drop and accepts noise within tolerance.

Non-goals:

- Do not use LLM judges for core recall regression gates unless the dataset explicitly requires judged answer quality.
- Do not let self-correction mutate production memory without review.

---

## Cross-cutting Rules

- Preserve Goncho's product thesis: **trust-preserving context architecture for long-horizon agents**.
- Core library stays embedded, local-first, and dependency-light.
- Server mode and broad integrations are adapters, not mandatory runtime requirements.
- Retrieval evidence must remain reproducible: no answer hints, no hidden LLM judges, no content-only scoring when stable IDs are required.
- Dangerous operations need explicit owner action: git writes, deletes, deploys, public binds, and distributed leases.
- Every new product surface needs a smoke command and a docs link.

---

## Suggested Implementation Order

1. **P0.1** `cmd/goncho-server serve/health` over current SQLite service. ✅
2. **P0.2** `make server-smoke` with write/search/recall/context. ✅
3. **P0.3** `goncho-server init/demo/doctor` plus minimal MCP-compatible `/mcp`. ✅
4. **P1.1** host hook schema package + privacy/redaction + docs guards. ✅
5. **P1.2** `goncho connect gormes` and one external connector dry-run.
6. **P2.1** read-only JSON viewer API before UI assets.
7. **P2.2** session timeline/replay JSON from observations.
8. **P3.1** compatibility tool registry for delivered safe aliases.
9. **P4.1** local vector index behind `Config.VectorStore` with fake-vector tests.
10. **Backlog A.1** mem0-style tiny facade over Goncho evidence APIs.
11. **Backlog D.1** MCP protocol compliance/resources/prompts hardening.
12. **Backlog F.1** onboarding/connect/remove/preference UX.
13. **Backlog B.1** conversation-to-memory extraction proposals.
14. **Backlog G.1** provider resilience and fallback diagnostics.
15. **Backlog H.1** disk-budget and retention preview.
16. **Backlog J.1** portable JSONL/Markdown export-import.
17. **Backlog K.1** eval feedback and regression gates.
18. **P5.1** server-mode threat model and auth requirements.
19. **P6.1** connector docs and doctor/upgrade-check polish.

---

## Current State Snapshot

What Goncho already has:

- Embedded Go SDK and service API.
- SQLite local storage and migrations.
- First-class `cmd/goncho-server` with `init`, `serve`, `health`, `demo`, `doctor`, minimal `/mcp`, and `make server-smoke`.
- Scoped search, recall, context packs, and provenance traces.
- Host-neutral hook capture API with P1 event schemas, redaction/truncation filters, and capture/permission docs.
- Optional vector-store seam and semantic RRF fusion.
- Query expansion provenance.
- Memory slots.
- Four-tier explicit consolidation API.
- Local action graph and signals.
- Snapshot manifests/diffs/rollback metadata.
- Image refs/checksums/metadata.
- Public tools: context, search, recall, remember, review, handoff.
- Memorymirror source-pinned architecture map against agentmemory.
- Deterministic LongMemEval-S, LOCOMO, BEAM, and backend-comparison benchmark harnesses.

What Goncho still lacks versus agentmemory:

- Polished standalone server release packaging beyond the first `goncho-server` runtime.
- Browser viewer and session replay UX.
- One-command broad agent connector/install flows.
- Automatic hook installation.
- Large documented compatibility MCP catalog.
- Bundled local embedding runtime and image embeddings.
- Distributed leases/signals/team sync.
- Transcript import UX.
- Operational deployment packaging.
- Top-level `goncho doctor`, `version --json`, and `upgrade-check` polish beyond `goncho-server doctor`.
- Connector-breadth docs and marketplace polish.
- Mem0-simple API facade with stable IDs, metadata filters, and history.
- Onboarding/remove/preference UX and portable export formats.
- Provider resilience, disk-budget retention, and eval feedback loops.

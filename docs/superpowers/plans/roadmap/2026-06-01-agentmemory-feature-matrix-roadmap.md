# Agentmemory Feature-Matrix Roadmap

**Date:** 2026-06-01

**Source evidence:** `docs/opensource-memory-systems/agentmemory` cloned from `https://github.com/rohitg00/agentmemory`; local worktree observed at `9b18a80` with fetched `origin/main` at `fd9e3bd`. Goncho evidence comes from `README.md`, `TODO.md`, `CONTEXT.md`, `memorymirror`, `cmd/goncho`, `cmd/goncho-server`, `http`, `service`, and existing docs guards.

**Goal:** Close the product-UX gaps surfaced by the Goncho × agentmemory feature matrix while preserving Goncho's trust-preserving, local-first core. Do not clone agentmemory wholesale. Convert broad upstream ideas into explicit Goncho seams, non-mutating plans, reviewable adapters, and testable operator surfaces.

## Shared constraints

- Keep the embedded Go service usable without a standalone server.
- Keep connector mutation behind explicit `--apply`; plan-only output is the default until smoke coverage exists.
- Keep broad MCP compatibility behind opt-in adapter/catalog surfaces; do not bloat the core tool contract.
- Keep retrieval changes benchmark-safe: no answer hints, no LLM judges, no benchmark gold-ID tuning, and no frozen artifact regeneration unless named by a plan.
- Keep team/distributed coordination behind explicit server-mode governance.
- Every slice names a focused public-interface test before broad validation.

## Feature matrix to delivery map

| Area | Goncho status | Best improvement | Owning seam | Priority |
| --- | --- | --- | --- | --- |
| Runtime/server | Partial parity | Polish first-run packaging/docs | `cmd/goncho-server`, docs-site, Makefile | P1 |
| Connectors | Gap | Add plan-only Cursor/Gemini/Hermes/OpenCode adapters | `cmd/goncho`, `internal/hostintegration` | P0 |
| Hook capture | Partial parity | Generated hook bundles with golden tests | `internal/hostintegration`, `service.CaptureHostHook` | P0 |
| MCP tools | Intentionally smaller | Opt-in compatibility catalog, not core sprawl | `memorymirror`, `toolmeta`, `cmd/goncho-server` | P1 |
| Retrieval | Strong, gaps | Local embedding/reindex + synonym provenance | `service`, `cmd/goncho`, `cmd/goncho-bench` | P1 |
| Viewer/replay | Gap | Recall trace + orientation-pack viewer | `http`, `service.Recall`, docs-site | P0 |
| Governance | Strong internals, less surfaced | Retention/access reports + sentinels | `service`, `internal/memorypolicy`, `http` | P1 |
| Team/leases | Deferred/partial | Keep behind explicit server mode | `service`, `policy`, `cmd/goncho-server security` | P2 |
| Portability | Strong | Add top-level `goncho export/import` CLI | `cmd/goncho`, `service` portable import/export | P1 |

## Phase 0 — Connector and viewer tracer bullets

### Slice 0.1 — Plan-only external connector adapters

**Goal:** Match agentmemory's connector breadth without silent host mutation.

**Files/modules:**

- `cmd/goncho/main.go`
- `internal/hostintegration/config`
- `internal/hostintegration/mapping`
- `internal/hostintegration/policy`
- `docs/integrations/deferred/*`
- `docs/guards/integrations`

**Behavior:**

- Add `goncho connect cursor --plan` and `goncho remove cursor --plan` for MCP JSON config snippets.
- Add `goncho connect gemini-cli --plan` and matching remove plan for MCP JSON config snippets.
- Add `goncho connect hermes --plan` and matching remove plan with explicit Gormes/Hermes handoff wording.
- Add `goncho connect opencode --plan` and matching remove plan using OpenCode's top-level `mcp` shape and hook-bundle placeholders.
- Keep `--apply` rejected until generated artifacts have host-level smoke coverage.

**Focused tests first:**

- `go test ./cmd/goncho -run TestConnectCursorPlanPrintsMCPPatch -count=1`
- `go test ./cmd/goncho -run TestConnectGeminiCLIPlanPrintsMCPPatch -count=1`
- `go test ./cmd/goncho -run TestConnectHermesPlanNamesGormesHandoff -count=1`
- `go test ./cmd/goncho -run TestConnectOpenCodePlanUsesTopLevelMCPShape -count=1`
- `go test ./cmd/goncho -run TestRemovePlansMirrorConnectPlans -count=1`

**Acceptance:**

- `go test ./cmd/goncho ./internal/hostintegration ./docs/guards/integrations -count=1`
- `go test ./...`
- `git diff --check`

### Slice 0.2 — Recall trace viewer JSON

**Goal:** Make Goncho's trust-preserving recall evidence visible before building a browser UI.

**Files/modules:**

- `http/routes.go`
- `http/service_handler.go`
- `service/goncho_recall_tool_output.go`
- `service/recall*`
- `docs-site/src/content/docs/operators/runbook.md`

**Behavior:**

- Add read-only `GET /v3/workspaces/{workspace}/viewer/recall?peer=&query=&session=&limit=`.
- Return trace ID, selected candidates, rejected candidates, provenance, graph relation paths, warnings, query expansion notes, and compact diagnostics.
- Do not add write operations to the viewer.

**Focused tests first:**

- `go test ./http -run TestViewerRecallEndpointReturnsTraceEvidence -count=1`
- `go test ./service -run TestRecallViewerPayloadIncludesRejectedAndWarnings -count=1`

**Acceptance:**

- `go test ./http ./service -count=1`
- `go test ./...`
- `git diff --check`

## Phase 1 — Hook bundles, compatibility catalog, and first-run polish

### Slice 1.1 — Generated hook bundles with golden tests

**Goal:** Make automatic capture reviewable without auto-installing it.

**Files/modules:**

- `internal/hostintegration/contracts`
- `internal/hostintegration/mapping`
- `cmd/goncho`
- `service/hook_capture.go`
- `service/internal/hookcapture`

**Behavior:**

- Introduce a `HookBundlePlan` value type that lists event, command, payload schema, redaction class, output path, and install status.
- Generate bundle plans for supported events: prompt, assistant response, pre-tool, post-tool, tool failure, pre-compact, stop, session end, subagent start/stop.
- Emit host-specific bundles only as plan JSON/markdown until smoke coverage exists.

**Focused tests first:**

- `go test ./internal/hostintegration -run TestHookBundlePlanMapsEverySupportedEvent -count=1`
- `go test ./cmd/goncho -run TestConnectCodexPlanIncludesHookBundleManifest -count=1`
- `go test ./service -run TestGeneratedHookPayloadsPassCaptureFilter -count=1`

**Acceptance:**

- `go test ./cmd/goncho ./internal/hostintegration ./service -count=1`
- `go test ./...`
- `git diff --check`

### Slice 1.2 — Opt-in agentmemory-compatible MCP catalog

**Goal:** Let hosts discover familiar `memory_*` operations without expanding Goncho's small core surface.

**Files/modules:**

- `memorymirror/tools.go`
- `memorymirror/architecture.go`
- `toolmeta`
- `cmd/goncho-server`
- `http` MCP handling

**Behavior:**

- Add `goncho-server serve -mcp-compat=core|agentmemory|off` or equivalent config.
- Keep default Goncho tools unchanged.
- In `agentmemory` mode, expose aliases backed by Goncho seams and mark partial/deferred tools as descriptors only unless executable.
- Include `source`, `status`, `goncho_seam`, `mutating`, `prompt_safe`, and `audit_kind` in catalog metadata.

**Focused tests first:**

- `go test ./memorymirror -run TestAgentmemoryCompatibilityCatalogMarksPartialTools -count=1`
- `go test ./cmd/goncho-server -run TestMCPCompatFlagKeepsDefaultToolSurfaceSmall -count=1`

**Acceptance:**

- `go test ./memorymirror ./cmd/goncho-server ./http -count=1`
- `go test ./...`
- `git diff --check`

### Slice 1.3 — First-run packaging and server polish

**Goal:** Make `goncho-server` feel product-ready while preserving explicit local-first defaults.

**Files/modules:**

- `cmd/goncho-server`
- `cmd/goncho`
- `Makefile`
- `docs-site/src/content/docs/start/quick-start.md`
- `docs-site/src/content/docs/operators/runbook.md`

**Behavior:**

- Add golden-tested first-run output that names DB path, config path, loopback bind, MCP URL, viewer endpoints, and next commands.
- Add `goncho-server doctor --server-url` to report a running server's health alongside local DB checks.
- Add release smoke coverage for first-run docs snippets.

**Focused tests first:**

- `go test ./cmd/goncho-server -run TestOnboardingOutputNamesServerAndViewerURLs -count=1`
- `go test ./cmd/goncho-server -run TestDoctorCanReadRunningServerHealth -count=1`

**Acceptance:**

- `go test ./cmd/goncho-server ./cmd/goncho ./docs/guards/release -count=1`
- `make server-smoke`
- `go test ./...`
- `git diff --check`

## Phase 2 — Retrieval and viewer UX depth

### Slice 2.1 — Local embedding reindex and diagnostics CLI ✅

**Goal:** Productize Goncho's optional vector seam without making embeddings mandatory.

**Files/modules:**

- `service/vector*`
- `service/provider_resilience.go`
- `cmd/goncho`
- `cmd/goncho-bench`
- `docs-site/src/content/docs/reference/retrieval-benchmarks.md`

**Behavior:**

- Added `goncho embeddings diagnose --db --index` with dimensions, count, stale rows, checksum, last indexed time, and missing/stale preview.
- Added `goncho embeddings reindex --plan|--dry-run` plus explicit `--apply`; local index path is explicit via `--index` or defaults to `<db>.vectors.json`.
- Added semantic vector evidence metadata showing candidates entered the RRF-capable recall lane and lexical fallback remained active.

**Focused tests first:**

- `go test ./cmd/goncho -run TestRunEmbeddingsDiagnoseReportsLocalVectorIndexHealth -count=1`
- `go test ./service -run TestLocalVectorIndexPersistsDiagnosticsAndFeedsRecall -count=1`

**Acceptance:**

- `go test ./cmd/goncho ./service ./cmd/goncho-bench -count=1`
- `go test ./...`
- `git diff --check`

### Slice 2.2 — Synonym and alias provenance ✅

**Goal:** Improve recall transparently without benchmark-specific hacks.

**Files/modules:**

- `service/internal/queryexpand`
- `service/internal/searchtokens`
- `service/recall*`
- `cmd/goncho-bench`

**Behavior:**

- Added `Config.QueryAliases` for a small configurable alias/synonym table with configured-vs-built-in provenance in Recall/Search traces.
- Query expansion evidence already surfaces through diagnostics and viewer recall JSON; LOCOMO recall diagnostics now record expansion terms and alias sources.
- Empty alias config preserves built-in query expansion behavior.

**Focused tests first:**

- ✅ `go test ./service -run TestRecallQueryExpansionProvenanceExplainsAliasHit -count=1`
- ✅ `go test ./cmd/goncho-bench -run TestBenchmarkRecallRecordsExpansionProvenance -count=1`

**Acceptance:**

- `go test ./service ./cmd/goncho-bench -count=1`
- `go test ./...`
- `git diff --check`

### Slice 2.3 — Orientation-pack viewer ✅

**Goal:** Show why a fact entered prompt-ready context.

**Files/modules:**

- `http`
- `service.Context`
- `service/recall_projector*`
- `docs-site`

**Behavior:**

- Added read-only viewer JSON at `/v3/workspaces/{workspace_id}/viewer/orientation` with orientation-pack context, token budgets, inclusion/exclusion reasons, warnings, and source provenance.
- Added `ContextResult.InclusionReasons` so context projection records why each section was included or skipped.
- Kept the viewer endpoint read-only; POST/other mutation attempts remain unregistered.

**Focused tests first:**

- ✅ `go test ./http -run TestViewerOrientationEndpointExplainsIncludedContext -count=1`
- ✅ `go test ./service -run TestContextProjectionRecordsInclusionReasons -count=1`

**Acceptance:**

- `go test ./http ./service -count=1`
- `cd docs-site && npm run build`
- `go test ./...`
- `git diff --check`

## Phase 3 — Governance and portability polish

### Slice 3.1 — Retention/access reports ✅

**Goal:** Turn trust internals into operator-facing stewardship.

**Files/modules:**

- `service/retention.go`
- `service/audit.go`
- `internal/memorypolicy`
- `http`
- `cmd/goncho`

**Behavior:**

- Added read-only `RetentionAccessReport` classifications for least-used, stale, high-risk, oversized, unreviewed, and over-budget memories.
- Added `goncho report retention --json` and read-only `/v3/workspaces/{workspace_id}/viewer/retention` JSON endpoint.
- Report is explicitly non-mutating; no deletion/archive happens from this path.

**Focused tests first:**

- ✅ `go test ./service -run TestRetentionAccessReportClassifiesStaleAndOversizedMemories -count=1`
- ✅ `go test ./cmd/goncho -run TestRetentionReportCommandIsReadOnly -count=1`

**Acceptance:**

- `go test ./service ./cmd/goncho ./http -count=1`
- `go test ./...`
- `git diff --check`

### Slice 3.2 — Sentinels and facets ✅

**Goal:** Warn when important facts are contradicted, stale, or absent from context without stuffing every sentinel into prompts.

**Files/modules:**

- `service`
- `internal/memoryannotations`
- `internal/memorypolicy`
- `http`

**Behavior:**

- Added reviewed sentinel records with scope, peer, expected condition, and active/reviewed status.
- Added lightweight memory facets and viewer memory filtering by facet/value.
- Surfaced sentinel warnings in `Context` and `Recall` when expected facts are relevant but absent.

**Focused tests first:**

- ✅ `go test ./service -run TestContextSentinelWarnsWhenImportantFactMissing -count=1`
- ✅ `go test ./service -run TestFacetFilterNarrowsViewerMemoryReport -count=1`

**Acceptance:**

- `go test ./service ./http -count=1`
- `go test ./...`
- `git diff --check`

### Slice 3.3 — Top-level export/import CLI ✅

**Goal:** Make existing portable formats accessible without writing Go code.

**Files/modules:**

- `cmd/goncho`
- `service/portable_export.go`
- `service/portable_import.go`
- `docs-site/src/content/docs/operators/runbook.md`

**Behavior:**

- Added `goncho export --db --workspace --profile --out --format jsonl|markdown --redaction-policy`.
- Added `goncho import preview --db --in` with counts, conflicts, schema version, redaction summary, and stable-ID collision handling.
- Added `goncho import apply --db --in --confirm APPLY`; apply refuses to run without explicit confirmation.

**Focused tests first:**

- ✅ `go test ./cmd/goncho -run TestExportCommandWritesPortableManifest -count=1`
- ✅ `go test ./cmd/goncho -run TestImportPreviewReportsConflictsWithoutWrites -count=1`
- ✅ `go test ./cmd/goncho -run TestImportApplyRequiresConfirm -count=1`

**Acceptance:**

- `go test ./cmd/goncho ./service -count=1`
- `go test ./...`
- `git diff --check`

## Phase 4 — Server-mode team/leases guardrail

### Slice 4.1 — Server-mode capability gate for leases/signals ✅

**Goal:** Avoid accidental distributed coordination in local embedded mode.

**Files/modules:**

- `cmd/goncho-server security`
- `service/action_leases.go`
- `policy`
- `docs/operations/server-mode-threat-model.md`

**Behavior:**

- Added machine-readable server-mode capability report with `local-only`, `team-preview`, and `team-enabled` modes.
- Distributed action leases now require explicit `server_mode=team-enabled`; local embedded mode rejects lease coordination.
- Local action graph operations remain available separately from distributed leases.

**Focused tests first:**

- ✅ `go test ./cmd/goncho-server -run TestSecurityReportMarksLeasesServerModeOnly -count=1`
- ✅ `go test ./service -run TestDistributedLeaseAPIsRejectWithoutServerMode -count=1`

**Acceptance:**

- `go test ./cmd/goncho-server ./service ./policy -count=1`
- `go test ./...`
- `git diff --check`

## Recommended execution order

1. Slice 0.1 connector plans.
2. Slice 0.2 recall trace viewer JSON.
3. Slice 1.1 hook bundle plans.
4. Slice 1.2 compatibility MCP catalog.
5. Slice 1.3 first-run/server polish.
6. Slice 2.1 vector diagnose/reindex dry-run.
7. Slice 2.2 synonym provenance.
8. Slice 2.3 orientation viewer.
9. Slice 3.1 retention/access reports.
10. Slice 3.3 top-level export/import CLI.
11. Slice 3.2 sentinels/facets.
12. Slice 4.1 server-mode lease/signal gate.

## Closeout validation for each completed slice

Run the focused tests named above, then:

```sh
go test ./...
git diff --check
```

For docs-site or public docs slices, also run:

```sh
cd docs-site && npm run build
```

For server runtime slices, also run:

```sh
make server-smoke
```

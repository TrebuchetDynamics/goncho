# Provider Landscape Lessons — Goncho Mega Improvement Plan

> **For agentic workers:** implement with Goncho TDD discipline. Each checkbox is a vertical slice: write the named failing test first, make the smallest service/API/UI change, then run the focused command and `go test ./...` when the slice touches public behavior.

**Goal:** Turn the Reddit memory-provider discussion into a concrete Goncho roadmap that improves perceived setup, speed, recall quality, inspectability, and trust without copying cloud/vendor lock-in or weakening Goncho's evidence-first model.

**Thesis:** The post's strongest signal is not "copy Mnemosyne." It is that users reward a memory layer that is local, installable in minutes, fast, inspectable, and good enough by default. Goncho already has the trust architecture; the mega improvement is to make that architecture feel as easy and observable as the best lightweight providers while keeping stronger provenance, review, temporal validity, and negative memory.

---

## Source Evidence

User-provided Reddit thread, summarized signals:

- Cloud providers are rejected for vendor lock-in and data retention concerns.
- Hindsight is respected for memory quality but criticized for weight, call volume, hidden config, setup complexity, and failure modes.
- OpenViking and Hancho are criticized for setup friction.
- Mnemosyne is praised for easiest setup, local SQLite, low latency, optional small local LLM, good quality/speed balance, BEAM 100K score, import paths, Hermes-native plugin, and dashboard ecosystem.
- Commenters highlight architecture ideas worth stealing: sqlite-vec + FTS5, multi-strategy recall, temporal triples/version chains, veracity tiers/conflict handling, sleep consolidation, memory banks, DeltaSync, PII-safe diagnostics, dashboard/replay, and Hindsight-as-L2/local-L1 cache layering.
- Signet/agentmemory are cited for cross-harness portability, raw transcript/source layer, repair tools, RBAC/team visibility, session replay, hooks, and MCP breadth.

External sources inspected during planning:

- Mnemosyne clone at `AxDSan/mnemosyne@c8e2953e1ac3abfcfb30ae99386705c2fc62a2de`:
  - `README.md`
  - `docs/architecture.md`
  - `docs/beam-benchmark.md`
  - `docs/comparison.md`
  - `mnemosyne/core/beam.py`
  - `mnemosyne/core/polyphonic_recall.py`
  - `mnemosyne/core/veracity_consolidation.py`
  - `mnemosyne/core/triples.py`
  - `mnemosyne/core/banks.py`
  - `mnemosyne/core/streaming.py`
  - `hermes_memory_provider/plugin.yaml`
- Signet README at `Signet-AI/signetai@74274d01b5619145821ae9369725e1e8622b4696`.

Goncho sources inspected:

- `README.md`
- `TODO.md`
- `docs/comparison.md`
- `docs/opensource-memory-systems/METAANALYSIS-MEMORY-SYSTEMS.md`
- `docs/benchmarks/ROADMAP.md`
- `docs-site/src/content/docs/start/current-capabilities.md`
- `docs-site/src/content/docs/roadmap/architecture-direction.md`
- `memorymirror/architecture.go`
- `memorymirror/backlog.go`
- `cmd/goncho/README.md`
- `cmd/goncho-server/README.md`
- `service/recall_pipeline.go`
- `service/local_vector_index.go`
- `service/four_tier_consolidation.go`
- `internal/memoryannotations/annotations.go`

---

## What Goncho Should Adopt, Adapt, and Avoid

### Adopt

- **Five-minute proof of value:** one copy-paste command that starts local memory, connects one host, writes one fact, recalls it, and opens/prints an inspector link.
- **First-class local embeddings:** optional bundled local embedding runtime plus reindex command; no hosted API required.
- **Polyphonic recall as a trace shape:** lexical/vector/graph/fact/temporal voices with deterministic fusion, per-voice scores, and ablation knobs.
- **Structured temporal facts:** stable hash IDs for subject-predicate-object facts, valid windows, version chains, and conflict review.
- **Sleep/consolidation UX:** explicit session-boundary consolidation that emits reviewable proposals, not silent truth.
- **Inspector/dashboard:** make recall, context packs, rejected candidates, graph paths, redaction, and lifecycle state legible.
- **Provider import/benchmark harnesses:** import from popular providers and compare with reproducible artifacts rather than marketing claims.

### Adapt

- **Mnemosyne BEAM tiers → Goncho tiers:** keep Goncho's evidence/claim/belief model; add working hot tier, episodic/session tier, semantic facts, procedural lessons, and scratchpad projections without making scratchpad durable truth.
- **Veracity tiers → Goncho trust model:** use source authority, evidence kind, confidence, review state, and stale/live-check warnings instead of a single global confidence score.
- **Memory banks → Goncho scope:** map banks to workspace/profile/agent/team scopes, not separate ad hoc SQLite files unless used for local L1 cache isolation.
- **DeltaSync → server-mode receipts:** incremental promotion/sync belongs behind explicit ACL, lease, and audit controls.

### Avoid

- Mandatory Python runtime, mandatory cloud calls, or model downloads during core library use.
- Silent LLM-extracted truth writes.
- A giant default MCP tool catalog that bypasses review/audit.
- Benchmark claims without pinned datasets, checksums, leakage controls, and failure audits.
- Public bind/team sync before auth and server-mode governance are boring.

---

## Mega Improvement Tracks

## Track 1 — Product Feel: "Goncho Works in Five Minutes"

**Why:** Reddit users chose the provider that was easiest to set up and inspect, not necessarily the theoretical best.

**Current Goncho status:** `goncho-server init/serve/health/demo/doctor`, top-level `goncho doctor/version/upgrade-check/preferences`, and non-mutating connector plans exist. The missing product moment is a single guided proof that applies no hidden mutations and shows the inspector/next commands.

### Slices

- [ ] **1.1 Add `goncho quickstart --plan` and `--local-demo`**
  - Behavior: prints or runs a non-destructive local proof: DB path, server command, MCP URL, demo write/recall/context proof, viewer URL, and host connect next steps.
  - First failing test: `TestQuickstartLocalDemoReportsWriteRecallContextAndViewerURL` in `cmd/goncho/main_test.go`.
  - Likely files: `cmd/goncho/main.go`, `cmd/goncho/README.md`, `README.md`, docs-site quick start.
  - Validation: `go test ./cmd/goncho ./cmd/goncho-server ./http ./service`.

- [ ] **1.2 Add connector smoke receipts for Hermes/Pi/Codex plans**
  - Behavior: each connector plan includes an executable smoke checklist and expected health/recall response shape.
  - First failing test: `TestConnectPlanIncludesSmokeReceipt` in `cmd/goncho/main_test.go`.
  - Non-goal: no auto-editing host config until `--apply` has host-specific golden tests.

- [ ] **1.3 Add README "two-session proof"**
  - Behavior: public docs mirror the Reddit expectation: remember one fact, restart/session switch, recall/context prove continuity.
  - First failing test: docs guard that checks README contains the two-session proof commands.

---

## Track 2 — Local Semantic Runtime: "Fast by Default, Better When Opted In"

**Why:** Mnemosyne's sqlite-vec/FTS5/local embedding story is the central perceived quality-speed win.

**Current Goncho status:** `TextEmbeddingProvider`, `LocalVectorIndex`, vector diagnostics, optional `Config.VectorStore`, and semantic RRF fusion exist. Missing pieces are a real Go-friendly embedding provider, reindex/import command, and clear install/diagnostic UX.

### Slices

- [ ] **2.1 Add `goncho embeddings reindex --plan` and service reindex API**
  - Behavior: previews count/checksum/stale rows for conclusions, observations, and selected imported docs; `--apply` writes only when explicit.
  - First failing test: `TestEmbeddingReindexPreviewDoesNotMutateAndReportsStaleRows` in `service/local_vector_index_test.go` or `cmd/goncho/main_test.go`.
  - Likely files: `service/local_vector_index.go`, new `service/vector_reindex.go`, `cmd/goncho/main.go`, docs.
  - Validation: `go test ./service -run 'Vector|Reindex'`.

- [ ] **2.2 Add optional bundled local embedding provider**
  - Behavior: a provider interface implementation with explicit model path/version/checksum and no implicit network download in core tests.
  - First failing test: `TestLocalEmbeddingProviderReportsModelChecksumAndDimensions`.
  - Non-goal: do not require Python, GPU, hosted APIs, or network at runtime.

- [ ] **2.3 Add sqlite-vec storage adapter investigation spike**
  - Behavior: prototype whether `sqlite-vec` can be optional behind `VectorStore` without destabilizing `ncruces/go-sqlite3` builds.
  - Success signal: ADR or prototype notes with build matrix, not production code unless the seam is safe.

---

## Track 3 — Polyphonic Recall Trace

**Why:** Reddit and Mnemosyne emphasize multi-strategy recall. Goncho already has the right philosophy; users need to see each retrieval voice and why fusion picked a result.

**Current Goncho status:** `RecallTrace` has candidates/selected/rejected/warnings; scoring includes keyword, semantic, graph, fact, recency, importance, and scope. Missing: voice-level candidate groups, ablation controls, and inspector-ready per-voice diagnostics.

### Slices

- [ ] **3.1 Add recall voice diagnostics**
  - Behavior: `RecallTrace` exposes voice summaries for lexical, vector, graph, fact, temporal, and lifecycle/trust adjustments.
  - First failing test: `TestRecallTraceIncludesPerVoiceDiagnostics` in `service/recall_pipeline_test.go`.
  - Likely files: `service/recall_ir.go`, `service/recall_pipeline.go`, `service/recall_diagnostics.go`, projector/viewer docs.

- [ ] **3.2 Add per-voice ablation config for benchmarks**
  - Behavior: benchmark configs can disable vector/graph/fact/temporal voices and write paired outcomes/failure deltas.
  - First failing test: `TestBenchRecallVoiceAblationWritesConfigIDAndFailureDeltas` in `cmd/goncho-bench`.
  - Validation: `make bench-beam-smoke`, `make bench-longmemeval-s-smoke`.

- [ ] **3.3 Add budget-aware diversity explanation**
  - Behavior: rejected candidates name duplicate/coverage/token-budget reasons in user-readable form for viewer and MCP resources.
  - First failing test: `TestRecallRejectedCandidatesExplainDiversityAndBudget`.

---

## Track 4 — MEMORIA-Style Temporal Fact Engine, Goncho-Style Trust

**Why:** Mnemosyne's biggest BEAM gains came from structured fact triples, temporal windows, gap analysis, and version chains. Goncho has fact annotations and graph recall, but not a first-class temporal SPO fact store with single-current-truth semantics.

**Current Goncho status:** deterministic fact annotations live in `goncho_memory_annotations`; contradiction detection exists; recall graph uses annotations. Missing: canonical SPO facts with stable IDs, valid intervals, supersession chains, and explicit distinction between append-only annotations and single-current facts.

### Slices

- [ ] **4.1 Add temporal fact store API**
  - Behavior: `AddTemporalFact(subject,predicate,object,valid_from,source,evidence)` closes prior current facts with same scoped `(subject,predicate)` and preserves history.
  - First failing test: `TestTemporalFactSupersedesPriorCurrentTruthButKeepsAsOfHistory`.
  - Likely files: new `service/temporal_facts.go`, migrations, export/import, audit.
  - Required trust rule: every fact has scope, source authority, evidence IDs, confidence, status, and review state.

- [ ] **4.2 Stable fact IDs with collision/smuggling tests**
  - Behavior: SHA-256 length-prefixed IDs over normalized `(scope,subject,predicate,object)`; no truncation collisions.
  - First failing test: `TestTemporalFactIDLengthPrefixedHashAvoidsTruncationAndSeparatorCollisions`.

- [ ] **4.3 Recall temporal facts as a separate voice**
  - Behavior: current-truth queries prefer current facts; as-of queries can retrieve historical facts with warnings when memory is stale/superseded.
  - First failing test: `TestRecallAsOfUsesHistoricalTemporalFactWhileCurrentTruthUsesLatest`.

- [ ] **4.4 Conflict review, not auto-truth**
  - Behavior: lower-confidence contradictory facts enter review; context pack shows conflict/stale warnings rather than silently choosing.
  - First failing test: `TestConflictingTemporalFactRoutesToReviewWhenAuthorityIsAmbiguous`.

---

## Track 5 — Sleep, Working Memory, and Scratchpad Without Context Poisoning

**Why:** Mnemosyne's `sleep()` and scratchpad are simple mental models. Goncho has four-tier consolidation, session summaries, memory slots, and dream scheduler pieces, but the operator UX is not yet as obvious.

**Current Goncho status:** `ExecuteFourTierConsolidation` exists but is deterministic/simple and writes conclusions directly. Extraction proposals and review routing exist elsewhere.

### Slices

- [ ] **5.1 Convert consolidation into proposal-first `goncho sleep`**
  - Behavior: session-boundary consolidation produces working/episodic/semantic/procedural proposals with provenance and review status; no silent canonical promotion.
  - First failing test: `TestSleepConsolidationProducesReviewableTieredProposals`.
  - Likely files: `service/four_tier_consolidation.go`, extraction proposals, review queue, `cmd/goncho`.

- [ ] **5.2 Add scratchpad as explicitly ephemeral memory**
  - Behavior: session-scoped scratchpad entries are not searched or exported as durable memory unless promoted through review.
  - First failing test: `TestScratchpadIsExcludedFromDefaultRecallAndPortableExportUntilPromoted`.
  - Likely files: new `service/scratchpad.go`, MCP/tool alias if safe, docs.

- [ ] **5.3 Add anti-bloat decay report for hot working memory**
  - Behavior: reports oversized/noisy/stale hot-tier candidates and proposed archive/promote actions.
  - First failing test: `TestWorkingMemoryDecayReportExplainsCandidatesWithoutDeleting`.

---

## Track 6 — Inspector, Dashboard, and Repair UX

**Why:** Users praise dashboards because they can see what was stored and why recall worked or failed. Signet's strongest lesson is raw-record fallback plus repair.

**Current Goncho status:** read-only viewer JSON and session timeline endpoints exist. Missing: browser UI, recall trace viewer, orientation-pack viewer, transcript import preview, and repair workflows in one operator surface.

### Slices

- [ ] **6.1 Recall trace viewer endpoint and static page**
  - Behavior: show candidates, selected/rejected, voice scores, provenance, warnings, query expansion, graph paths, and token budget.
  - First failing test: `TestViewerRecallTraceEndpointReturnsVoiceScoresAndRejectedReasons` in `http/viewer_test.go`.

- [ ] **6.2 Orientation pack viewer**
  - Behavior: show exactly what entered context, why, and which live verification warnings remain.
  - First failing test: `TestViewerOrientationPackExplainsIncludedWarningsAndProvenance`.

- [ ] **6.3 Repair inbox**
  - Behavior: operator can preview edit/supersede/archive/reclassify actions; apply requires explicit audited operation.
  - First failing test: `TestRepairPreviewDoesNotMutateAndNamesAuditOperation`.

- [ ] **6.4 Transcript/source fallback**
  - Behavior: every derived summary/fact in viewer links back to raw messages, observations, import records, or file checksums.
  - First failing test: `TestViewerDerivedMemoryLinksToRawEvidence`.

---

## Track 7 — Provider Import, Migration, and Competitive Bench Harness

**Why:** The Reddit thread is a buying comparison. Goncho should make migration easy and benchmark comparisons honest.

**Current Goncho status:** portable JSONL/Markdown export/import exists; BEAM-compatible artifacts and paired comparison exist; LOCOMO/LongMemEval backends exist. Missing: provider-specific importers for Mnemosyne/Hindsight/Signet/Hancho-style exports and current public comparison docs for the Reddit landscape.

### Slices

- [ ] **7.1 Add Mnemosyne export importer preview**
  - Behavior: import `working_memory`, `episodic_memory`, annotations/triples when present; preview counts, conflicts, scopes, redaction, and fact mapping before apply.
  - First failing test: `TestMnemosyneImportPreviewMapsWorkingEpisodicTriplesAndDetectsConflicts`.
  - Non-goal: do not depend on Mnemosyne Python code at runtime.

- [ ] **7.2 Add Hindsight export importer preview**
  - Behavior: preserve source IDs, banks/scopes, facts, graph edges, temporal metadata, and confidence labels where available.
  - First failing test: `TestHindsightImportPreviewPreservesStableIDsAndBankScopes`.

- [ ] **7.3 Add provider landscape comparison doc**
  - Behavior: compare Goncho, Mnemosyne, Hindsight, Signet, agentmemory by architecture, setup, storage, recall, trust, UX, team mode, and benchmark caveats.
  - First failing test: docs guard that comparison includes each provider and benchmark caveat.

- [ ] **7.4 Add real BEAM paired-import workflow docs**
  - Behavior: document how to import nested Mnemosyne `beam_e2e_results.json`, run Goncho on the same converted BEAM fixture, and compare paired outcomes with leakage checks.
  - Validation: `make bench-beam-smoke` plus a documented full-run command.

---

## Track 8 — Local L1 / Shared L2 Architecture

**Why:** Reddit's best architecture suggestion is local Mnemosyne-style L1 for speed plus Hindsight/server L2 for durability/team sharing. Goncho can own both sides if the boundary is explicit.

**Current Goncho status:** local SQLite embedded mode, server mode threat model, team feed, leases/signals, export/import, ACL pieces. Missing: productized L1/L2 promotion semantics and safe sync receipts.

### Slices

- [ ] **8.1 Define hot local L1 vs shared L2 contract**
  - Behavior: doc/API contract for what may stay local, what may be promoted, required redaction, ownership, ACL, and audit receipt.
  - First failing test: docs guard for L1/L2 contract and no public-bind sync without auth.

- [ ] **8.2 Promotion preview and receipt**
  - Behavior: local facts/episodes can be previewed for shared promotion with conflicts, redaction, provenance, and rollback metadata.
  - First failing test: `TestSharedPromotionPreviewRequiresACLAndWritesReceiptOnlyOnApply`.

- [ ] **8.3 Delta sync after server auth is real**
  - Behavior: incremental sync uses allowlisted tables/fields, destination-controlled scope/lifecycle, and replayable receipts.
  - Non-goal: no P2P mesh or peer mutation of lifecycle fields.

---

## Track 9 — Docs and Roadmap Hygiene

**Why:** Goncho's TODO and current-capabilities docs have some status drift. A trust-preserving memory project should keep its own roadmap trustworthy.

### Slices

- [ ] **9.1 Reconcile TODO current-state snapshot with delivered checklist**
  - Behavior: remove stale "still lacks" items that are already delivered, or mark them as polish gaps.
  - First failing test: `TestTodoCurrentStateDoesNotContradictDeliveredChecklist` docs guard.

- [ ] **9.2 Update memorymirror backlog statuses**
  - Behavior: backlog item statuses match shipped vector index, hook schemas, resources/prompts, slot APIs, leases/signals where delivered.
  - First failing test: `TestMemoryMirrorBacklogStatusesMatchArchitectureManifest`.

- [ ] **9.3 Add `docs/comparison.md` provider caveat expansion**
  - Behavior: includes Mnemosyne and Signet without star-count hype; every score has methodology caveat.

---

## Recommended Implementation Order

1. **Track 9.1 + 9.2** — roadmap hygiene so future agents do not plan from stale state.
2. **Track 1.1** — five-minute local proof, because adoption friction is the Reddit thread's loudest signal.
3. **Track 3.1** — per-voice recall diagnostics, because it improves quality work and viewer work at the same time.
4. **Track 2.1** — reindex preview, because local semantic runtime needs operator safety before real model support.
5. **Track 4.1 + 4.2** — temporal fact store with stable IDs, because it is the highest-quality architectural gap from Mnemosyne/MEMORIA.
6. **Track 6.1** — recall trace viewer, because inspectability turns trust architecture into product UX.
7. **Track 7.1** — Mnemosyne importer preview, because it answers the current market conversation directly.

---

## First TDD Slice to Start After Approval

**Chosen slice:** Track 9.1 — reconcile roadmap/status drift.

**Why this first:** It is safe, bounded, and prevents the next implementation agent from following stale TODO evidence. It also makes the mega roadmap easier to review before deeper code work.

**First failing test:** `TestTodoCurrentStateDoesNotContradictDeliveredChecklist` in a docs guard test. The test should parse or assert the specific contradictory statements around already-delivered capabilities such as top-level `goncho doctor`, `version --json`, `upgrade-check`, mem0-style facade, onboarding/preferences, portable export, provider resilience, disk-budget retention, and eval feedback loops.

**Likely files:**

- `docs/*_guard_test.go` or a new `docs/roadmap_status_guard_test.go`
- `TODO.md`
- possibly `memorymirror/backlog.go`

**Risks/non-goals:**

- Do not turn this into a broad rewrite of the roadmap.
- Do not mark capabilities complete without tests/evidence.
- Do not change product commitments; only resolve contradictions and add links to this mega plan.

**Approval gate:** After this plan is accepted, switch to Goncho TDD implementation and start with the failing docs guard above.

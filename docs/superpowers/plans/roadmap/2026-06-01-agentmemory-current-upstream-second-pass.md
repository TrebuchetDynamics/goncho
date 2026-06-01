# Agentmemory Current-Upstream Second Pass

**Date:** 2026-06-01

**Source evidence:** `docs/opensource-memory-systems/agentmemory` has `origin` set to `https://github.com/rohitg00/agentmemory`. The local checked-out branch remains `feature/stable-external-memory-ids` at `9b18a80`; after `git fetch origin`, current upstream is `origin/main` at `fd9e3bd` (`v0.9.24`). This pass compares `9b18a80..origin/main` and the already-shipped Goncho roadmap commit `2bd4381`.

**Goal:** Re-check agentmemory's current upstream after Goncho's first feature-matrix implementation. Keep Goncho local-first, explicit, trust-preserving, and benchmark-safe. Treat agentmemory as a donor of UX seams and validation ideas, not as an architecture to clone.

## Upstream changes since the first clone

The upstream delta is large: 186 files changed, including README refreshes, 11 translated READMEs, new benchmark/eval harnesses, more connector adapters, Copilot plugin assets, project/agent isolation fixes, hook hardening, graph/session-end fixes, provider diagnostics, and release/CI polish.

Notable upstream commits/features:

- `bfb5e66` — Qwen Code, Antigravity, and Kiro connect adapters.
- `047ed72` — Warp, Cline, Continue, Zed, and Droid connect adapters.
- `a0da02f` — GitHub Copilot CLI support and plugin wiring.
- `b8027c9` — prints `npx skills add rohitg00/agentmemory -y` after successful connect.
- `1aec56a` / `bda73cc` family — `AGENT_ID` multi-agent isolation and user-facing scoped recall behavior.
- `7fb72f4` — adapter-pluggable benchmark harness with an in-house coding-agent corpus.
- `7f3a757`, `e39d962`, `acb4567`, `e8d6a02f` — graph/vector/session hardening.
- `6d9bfed`, `db9f000`, `edd1ceb`, `bda73cc` — product polish around logs, non-TTY onboarding, splash/doctor hints.
- `bfde288`, `051d5ac`, README provider sections — explicit local/provider setup and provider-specific environment variables.

## Second-pass feature matrix

| Area | Current agentmemory upstream | Goncho after first pass | Remaining Goncho opportunity | Priority |
| --- | --- | --- | --- | --- |
| Connector breadth | 17 adapters: Claude Code, Copilot CLI, Codex, Cursor, Gemini, Qwen, Antigravity, Kiro, Warp, Cline, Continue, Zed, Droid, OpenClaw, Hermes, pi, OpenHuman | Plan-only Codex/pi/Gormes/Cursor/Gemini/Hermes/OpenCode | Add plan-only adapters for Copilot CLI, Qwen, Antigravity, Kiro, Warp, Cline, Continue, Zed, and Droid; keep `--apply` rejected | P0 |
| Skills onboarding | Connect success tells users to install 8 skills via `npx skills add` | Hook bundle manifests and connector plans exist, but skill hints are not first-class | Add plan-only skill-install hints/manifests keyed by host, without shelling out or mutating agent skill dirs | P1 |
| Project/agent isolation | `AGENT_ID` tags writes and `AGENTMEMORY_AGENT_SCOPE=isolated` filters recall/viewer endpoints | Goncho has workspace/profile/peer/session scope and `agent_id` storage seams, but no operator-facing env-mode parity | Add explicit CLI/server config for actor/agent role isolation and viewer filter evidence; avoid hidden env magic in embedded service | P0 |
| Benchmarking | Pluggable eval harness with grep/vector/agentmemory adapters and `coding-agent-life-v1` corpus | Goncho has deterministic Go LOCOMO/LongMemEval/BEAM/backends harnesses | Add a small real-agent-session replay fixture modeled after coding-agent-life, scored by stable IDs and trust diagnostics | P1 |
| Graph hardening | Session-end triggers graph extraction; parser tolerates self-closing/reordered XML | Goncho graph recall exists, plus benchmark graph companions | Add graph health diagnostics for extraction freshness, orphan entities, and relation parser rejects; surface in viewer/doctor | P1 |
| Hook reliability | Fire-and-forget hooks, project basename normalization, task-completed hook, Copilot hooks | Goncho hook capture filters and bundle plans are reviewable | Add hook replay diagnostics and project-field normalization to prove host payloads will not leak absolute paths | P1 |
| Product polish | Non-TTY onboarding, better doctor hints, version flag, hard-pin enforcement, quiet logs | Goncho has onboarding, doctor, server smoke | Add `--version` smoke coverage and non-TTY/no-prompt contract tests; keep local server logs content-safe | P2 |
| Provider configuration | Separate embedding/chat base URLs and keys; local-model docs | Goncho has local hash embeddings, provider health diagnostics | Add docs/config examples for separate embedding vs summarization providers and local model profiles | P2 |
| Viewer UX | Memory sort newest-first, graph cooldown, memories pagination | Goncho has read-only recall/orientation/retention/memory JSON endpoints | Add API-level pagination/sort contracts for viewer memory report and graph freshness warnings | P1 |
| Maintenance/CI | Cross-platform CI, path ignores, release tags | Goncho validation already uses Go/docs/smoke discipline | Consider Windows-safe connector plan tests only; do not broaden CI without owner release decision | P3 |

## Recommended next slices

### Slice S2.1 — More plan-only connector adapters ✅

**Goal:** Match upstream adapter breadth without silent host mutation.

**Files/modules:**

- `cmd/goncho/main.go`
- `cmd/goncho/main_test.go`
- `internal/hostintegration`
- `docs/integrations/deferred/agent-hosts/*`
- `docs/guards/integrations`

**Behavior:**

- Added `goncho connect/remove <agent> --plan` for: `copilot-cli`, `qwen`, `antigravity`, `kiro`, `warp`, `cline`, `continue`, `zed`, and `droid`.
- Added host-specific config shapes where they matter:
  - Copilot CLI: `mcpServers` under Copilot MCP config.
  - Continue: array-form `mcpServers`; YAML existing-config mutation remains manual-plan-only.
  - Zed: `context_servers`, not `mcpServers`.
  - Warp/Cline/Droid/Qwen/Kiro/Antigravity: standard MCP JSON shapes with documented paths.
- Kept `--apply` rejected for all new adapters until host smoke coverage exists.

**Focused tests first:**

- ✅ `go test ./cmd/goncho -run TestConnectCopilotCLIPlanPrintsMCPServersPatch -count=1`
- ✅ `go test ./cmd/goncho -run TestConnectContinuePlanUsesArrayMCPServers -count=1`
- ✅ `go test ./cmd/goncho -run TestConnectZedPlanUsesContextServers -count=1`
- ✅ `go test ./cmd/goncho -run TestConnectSecondPassAgentsRejectApply -count=1`

**Acceptance:**

- `go test ./cmd/goncho ./internal/hostintegration ./docs/guards/integrations -count=1`
- `go test ./...`
- `git diff --check`

### Slice S2.2 — Explicit agent/role isolation mode ✅

**Goal:** Convert upstream `AGENT_ID` isolation into Goncho-native, trust-preserving role scope without surprising embedded users.

**Files/modules:**

- `service.Config`
- `service/search*`, `service/recall*`, `service/viewer.go`
- `cmd/goncho-server`
- `http/service_handler.go`

**Behavior:**

- Added explicit `AgentRoleID` and opt-in `AgentScopeMode` (`shared|isolated`) config fields.
- In isolated mode, service construction maps the observer scope to the configured role; recall/search/viewer emit agent-scope evidence.
- Preserved workspace/profile/peer/session scoping and kept embedded service config explicit.
- Did not read global env vars inside service constructors; `goncho-server` exposes explicit flags/config fields.

**Focused tests first:**

- ✅ `go test ./service -run TestRecallAgentScopeIsolationFiltersOtherRoles -count=1`
- ✅ `go test ./service -run TestViewerMemoryReportRecordsAgentScopeEvidence -count=1`
- ✅ `go test ./cmd/goncho-server -run TestServerAgentScopeConfigIsExplicit -count=1`

**Acceptance:**

- `go test ./service ./http ./cmd/goncho-server -count=1`
- `go test ./...`
- `git diff --check`

### Slice S2.3 — Real-agent replay microbenchmark ✅

**Goal:** Add a tiny, checked-in replay benchmark that measures practical agent memory utility without cloud dependencies or gold leakage.

**Files/modules:**

- `cmd/goncho-bench`
- `cmd/goncho-bench/testdata/*`
- `docs/benchmarks/ROADMAP.md`
- `docs-site/src/content/docs/roadmap/benchmark-roadmap.md`

**Behavior:**

- Added a small fictional coding-agent session corpus inspired by upstream `coding-agent-life-v1` but written for Goncho's existing benchmark contracts.
- Scored by stable inserted memory IDs, not answer text or LLM judgment.
- Included a replay smoke target that emits JSON and failure artifacts through the deterministic harness.
- Documented it as a smoke guardrail, not a publishable benchmark claim.

**Focused tests first:**

- ✅ `go test ./cmd/goncho-bench -run TestCodingAgentReplayScoresStableIDs -count=1`
- ✅ `make bench-agent-replay-smoke`

**Acceptance:**

- `go test ./cmd/goncho-bench -count=1`
- `make bench-agent-replay-smoke`
- `go test ./...`
- `git diff --check`

### Slice S2.4 — Graph and hook diagnostics ✅

**Goal:** Turn upstream hardening fixes into Goncho-visible health checks.

**Files/modules:**

- `service/recall_diagnostics.go`
- `service/hook_capture.go`
- `cmd/goncho-server doctor`
- `http` viewer diagnostics

**Behavior:**

- Added graph health diagnostics for relation count, orphan relation count, parser rejects, and freshness status.
- Added hook replay diagnostics that normalize project fields to non-sensitive basenames and prove absolute paths are not leaked in warnings.
- Surfaced diagnostics through `goncho-server doctor` as a first read-only health check.

**Focused tests first:**

- ✅ `go test ./service -run TestGraphDiagnosticsReportParserRejectsAndOrphans -count=1`
- ✅ `go test ./service -run TestHookDiagnosticsNormalizeProjectBasename -count=1`
- ✅ `go test ./cmd/goncho-server -run TestDoctorReportsGraphAndHookDiagnostics -count=1`

**Acceptance:**

- `go test ./service ./cmd/goncho-server ./http -count=1`
- `go test ./...`
- `git diff --check`

## Recommended execution order

1. S2.1 connector breadth: visible UX gap and low-risk because it remains plan-only.
2. S2.2 role isolation: high-value safety feature that aligns with Goncho's scoped memory model.
3. S2.3 replay benchmark: strengthens claims without benchmark shortcuts.
4. S2.4 graph/hook diagnostics: turns upstream reliability fixes into operator evidence.

## Completion note

This second pass found no need to replace the first Goncho roadmap. The first pass already closed the biggest gaps around MCP compatibility, trace viewers, embeddings, governance reports, import/export, and server-mode guardrails. Current upstream mainly adds breadth (more hosts), role isolation polish, benchmark/eval packaging, and operational hardening. Those should be implemented as bounded Goncho-native slices, not as direct ports.

# Goncho TODO

## Release state

- 2026-05-22: public module adoption smoke added for `github.com/TrebuchetDynamics/goncho@latest`.
  - Evidence target: `make public-module-smoke` creates a temporary external Go module, runs `go get github.com/TrebuchetDynamics/goncho@latest`, and compiles a minimal public API import.
  - Result: release readiness now separates library importability proof from the still-checkout-local benchmark CLI.

- 2026-05-22: public `@latest` still resolves to v0.1.0, so `go install github.com/TrebuchetDynamics/goncho/cmd/goncho-bench@latest` does not work yet.
  - Evidence: public install reports module `github.com/TrebuchetDynamics/goncho@latest` found at `v0.1.0`, but it does not contain `cmd/goncho-bench`.
  - Result: docs now point benchmark CLI users at checkout-local `make install-smoke` / `go install ./cmd/goncho-bench` until the next v0.1.x tag contains the command.

- 2026-05-22: generated primer/token-budget E2E coverage added for the public `goncho_context` tool.
  - Focused evidence: `go test . -run TestGonchoGoalPublicContextToolGeneratesPrimerWithinTokenBudgetE2E -count=1` passed.
  - Result: public context-tool coverage now proves generated orientation output preserves the newest in-budget turns and excludes older turns outside `max_tokens`.

- 2026-05-21 20:11 CST: stale `cmd/goncho-bench` expectation-drift blocker from 2026-05-20 is resolved on current `main`.
  - Focused evidence: `go test ./cmd/goncho-bench -run 'TestClassifyFailureCasesSelectsHardRanksAndCategories|TestWriteFailureCategoryReportsEmitsJSONLAndMarkdown' -count=1` passed.
  - Full Go evidence: `go test ./... -count=1` passed.
  - Result: benchmark classifier expectation drift no longer blocks Go verification.

- 2026-05-19: stale full-verification blocker resolved. Current release gate passes with:
  - `go test ./integration/gormes`
  - `go test ./...`
  - `cd docs-site && npm run build`

## Next roadmap items

- Continue lifecycle trust work: temporal validity, supersession chains, and confidence/freshness scoring.
- Expand graph/cognitive-map features behind deterministic tests.
- Add optional PostgreSQL/team adapter only after local SQLite API remains stable.

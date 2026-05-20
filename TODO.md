# Goncho TODO

## Release state

[BLOCKED] Full Goncho suite after skill-learning governance slice — 2026-05-20 10:08:28 CST
  blocker: `cmd/goncho-bench` expectation drift prevents claiming `go test ./... -count=1` green.
  evidence: `TestClassifyFailureCasesSelectsHardRanksAndCategories` got category `duplicate_near_duplicate_content`, want `temporal_ambiguity`; `TestWriteFailureCategoryReportsEmitsJSONLAndMarkdown` markdown missing `lexical_miss` and reports `direct_answer_mismatch`/`duplicate_near_duplicate_content`.
  unblocks when: bench classifier expectations or implementation are reconciled by the owning bench/retrieval work.
  owner: Goncho bench/retrieval owner.
  workaround/pivot: validate the skill-learning governance slice with focused root-package tests and non-bench packages, then leave bench failure untouched.
  next check: 2026-05-20 12:00 CST

- 2026-05-19: stale full-verification blocker resolved. Current release gate passes with:
  - `go test ./integration/gormes`
  - `go test ./...`
  - `cd docs-site && npm run build`

## Next after v0.1.0

- Add a generated primer/token-budget E2E.
- Continue lifecycle trust work: temporal validity, supersession chains, and confidence/freshness scoring.
- Expand graph/cognitive-map features behind deterministic tests.
- Add optional PostgreSQL/team adapter only after local SQLite API remains stable.

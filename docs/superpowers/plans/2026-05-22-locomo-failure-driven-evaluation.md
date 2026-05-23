# LOCOMO Failure-Driven Evaluation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add failure-audit buckets that turn LOCOMO misses into actionable wrong-branch and missing-companion evidence before any new retrieval tuning.

**Architecture:** Keep scoring centralized, deterministic, and stable-ID based. Extend the LOCOMO failure-audit classifier to label retrieval failures that already expose enough ID and conversation evidence: wrong branch retrieval and missing companion memories. The implementation writes audit labels only; it does not change retrieval, scoring, full-run artifacts, or benchmark gold data.

**Tech Stack:** Go, `cmd/goncho-bench`, JSONL LOCOMO failure audits, stable inserted `memory_id`, `go test`, existing release metadata smoke guards.

---

## Scope and guardrails

This plan converts the roadmap priority "Drive changes from failure-audit buckets such as missing candidates, rank-too-low candidates, wrong branch retrieval, and missing companion memories" into a focused implementation slice.

Hard constraints:

- Preserve frozen LOCOMO artifacts until a new date-stamped full run is intentionally generated.
- Keep LOCOMO scoring centralized and ID-based.
- Classify only from query metadata, retrieved stable IDs, gold stable IDs, ranks, and conversation IDs.
- Use stable inserted `memory_id` as the only evidence key for benchmark scoring and failure labels.
- Use no answer hints, no LLM judges, no answer-text scoring, and no benchmark-specific gold-ID hacks.
- Do not tune against LOCOMO gold IDs.
- Do not alter `docs/benchmarks/results/locomo-backend-comparison.json` or frozen full-run JSON in this slice.
- Do not change retrieval ranking or candidate generation in this slice.
- Add tests before production classifier changes.

## File structure

- Modify `cmd/goncho-bench/failure_classifier.go`: add a small LOCOMO failure-bucket classifier for wrong branch retrieval and missing companion memories.
- Modify `cmd/goncho-bench/failure_classifier_test.go`: prove bucket labels with stable-ID-only fixtures.
- Modify `cmd/goncho-bench/locomo.go`: thread the new bucket name into emitted LOCOMO failure-audit rows only after focused classifier tests pass.
- Modify `docs/benchmarks/ROADMAP.md`: record the delivered failure-audit bucket slice after behavior is proven.
- Modify `docs-site/src/content/docs/roadmap/benchmark-roadmap.md`: mirror the public roadmap note after behavior is proven.
- Modify `TODO.md`: record release-state evidence after validation.

## Task 1: Prove wrong-branch and missing-companion labels with a failing classifier test

**Files:**
- Modify: `cmd/goncho-bench/failure_classifier_test.go`
- Test: `cmd/goncho-bench/failure_classifier_test.go`

- [ ] **Step 1: Add the failing classifier test**

Append this test to `cmd/goncho-bench/failure_classifier_test.go`:

```go
func TestLocomoFailureAuditClassifiesWrongBranchAndMissingCompanionBuckets(t *testing.T) {
	cases := []struct {
		name string
		row  locomoFailureAuditRow
		want string
	}{
		{
			name: "wrong branch retrieval",
			row: locomoFailureAuditRow{
				QuestionID:      "locomo-conv-7-q-003",
				ConversationID:  "locomo-conv-7",
				GoldMemoryIDs:  []string{"locomo-conv-7-m-011"},
				TopMemoryIDs:   []string{"locomo-conv-8-m-002", "locomo-conv-8-m-004"},
				GoldRank:       -1,
				FailureReason:  "missing_candidate",
			},
			want: "wrong_branch_retrieval",
		},
		{
			name: "missing companion memories",
			row: locomoFailureAuditRow{
				QuestionID:      "locomo-conv-9-q-014",
				ConversationID:  "locomo-conv-9",
				GoldMemoryIDs:  []string{"locomo-conv-9-m-010", "locomo-conv-9-m-027"},
				TopMemoryIDs:   []string{"locomo-conv-9-m-010", "locomo-conv-9-m-040"},
				GoldRank:       1,
				FailureReason:  "partial_hit",
			},
			want: "missing_companion_memory",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyLocomoFailureBucket(tc.row); got != tc.want {
				t.Fatalf("classifyLocomoFailureBucket() = %q, want %q", got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run the focused classifier test to verify RED**

Run:

```bash
go test ./cmd/goncho-bench -run TestLocomoFailureAuditClassifiesWrongBranchAndMissingCompanionBuckets -count=1
```

Expected result: FAIL because `classifyLocomoFailureBucket` does not exist yet.

## Task 2: Add the minimal stable-ID-only failure-bucket classifier

**Files:**
- Modify: `cmd/goncho-bench/failure_classifier.go`
- Test: `cmd/goncho-bench/failure_classifier_test.go`

- [ ] **Step 1: Add the classifier helper**

Add this helper in `cmd/goncho-bench/failure_classifier.go` near existing failure classification helpers:

```go
func classifyLocomoFailureBucket(row locomoFailureAuditRow) string {
	if locomoFailureHasOutOfConversationTopHit(row) {
		return "wrong_branch_retrieval"
	}
	if locomoFailureHasMissingCompanion(row) {
		return "missing_companion_memory"
	}
	if row.FailureReason != "" {
		return row.FailureReason
	}
	return "unclassified_failure"
}

func locomoFailureHasOutOfConversationTopHit(row locomoFailureAuditRow) bool {
	if row.ConversationID == "" {
		return false
	}
	prefix := row.ConversationID + "-"
	for _, id := range row.TopMemoryIDs {
		if id != "" && !strings.HasPrefix(id, prefix) {
			return true
		}
	}
	return false
}

func locomoFailureHasMissingCompanion(row locomoFailureAuditRow) bool {
	if len(row.GoldMemoryIDs) < 2 {
		return false
	}
	retrieved := make(map[string]struct{}, len(row.TopMemoryIDs))
	for _, id := range row.TopMemoryIDs {
		if id != "" {
			retrieved[id] = struct{}{}
		}
	}
	matched := 0
	for _, id := range row.GoldMemoryIDs {
		if _, ok := retrieved[id]; ok {
			matched++
		}
	}
	return matched > 0 && matched < len(row.GoldMemoryIDs)
}
```

If `failure_classifier.go` does not already import `strings`, add it.

- [ ] **Step 2: Run the focused classifier test to verify GREEN**

Run:

```bash
go test ./cmd/goncho-bench -run TestLocomoFailureAuditClassifiesWrongBranchAndMissingCompanionBuckets -count=1
```

Expected result: PASS.

## Task 3: Thread the bucket into LOCOMO failure-audit rows

**Files:**
- Modify: `cmd/goncho-bench/locomo.go`
- Modify: `cmd/goncho-bench/locomo_test.go`

- [ ] **Step 1: Add a focused emitted-row test**

Add a test that writes a tiny LOCOMO failure audit with one multi-gold partial hit and checks the JSONL row includes:

```json
"failure_bucket":"missing_companion_memory"
```

The fixture must use only stable inserted `memory_id` values already present in the test memories map. Do not inspect answer text.

- [ ] **Step 2: Run the emitted-row test to verify RED**

Run the exact test name chosen in Step 1:

```bash
go test ./cmd/goncho-bench -run TestWriteLocomoFailureAuditEmitsFailureBucket -count=1
```

Expected result: FAIL because the JSON row does not yet include `failure_bucket`.

- [ ] **Step 3: Add the JSON field**

Add a `FailureBucket string` JSON field to the LOCOMO failure-audit row type and set it with:

```go
row.FailureBucket = classifyLocomoFailureBucket(row)
```

This must happen after stable ID validation and before JSON encoding.

- [ ] **Step 4: Run the emitted-row test to verify GREEN**

Run:

```bash
go test ./cmd/goncho-bench -run TestWriteLocomoFailureAuditEmitsFailureBucket -count=1
```

Expected result: PASS.

## Task 4: Document the delivered failure-audit bucket behavior after tests pass

**Files:**
- Modify: `docs/benchmarks/ROADMAP.md`
- Modify: `docs-site/src/content/docs/roadmap/benchmark-roadmap.md`
- Modify: `TODO.md`
- Modify: `release_metadata_test.go`
- Modify: `Makefile`

- [ ] **Step 1: Add a roadmap guard first**

Add `TestBenchmarkRoadmapSurfacesLocomoFailureDrivenEvaluationSlice` to `release_metadata_test.go` and make it require these markers in both roadmap files:

- `Failure-driven evaluation slice delivered`
- `TestLocomoFailureAuditClassifiesWrongBranchAndMissingCompanionBuckets`
- `wrong branch retrieval`
- `missing companion memories`
- `failure-audit buckets`
- `stable-ID memories`
- `without regenerating LOCOMO full-run artifacts`

Run it before editing roadmap docs:

```bash
go test . -run TestBenchmarkRoadmapSurfacesLocomoFailureDrivenEvaluationSlice -count=1
```

Expected result: FAIL until the roadmap note exists.

- [ ] **Step 2: Add internal and public roadmap notes**

Add this sentence after the speaker routing note in both roadmap files:

```markdown
Failure-driven evaluation slice delivered: `TestLocomoFailureAuditClassifiesWrongBranchAndMissingCompanionBuckets` proves wrong branch retrieval and missing companion memories can be separated into failure-audit buckets while preserving stable-ID memories without regenerating LOCOMO full-run artifacts.
```

- [ ] **Step 3: Wire the guard into release metadata smoke**

Add `BenchmarkRoadmapSurfacesLocomoFailureDrivenEvaluationSlice` to `make release-metadata-smoke` and to `TestReleaseMetadataSmokeIncludesLocomoResultDocsGuards`.

- [ ] **Step 4: Record release-state evidence**

Add this entry near the top of `TODO.md`:

```markdown
- 2026-05-22: LOCOMO failure-driven evaluation has its first implementation slice.
  - Evidence target: `go test ./cmd/goncho-bench -run TestLocomoFailureAuditClassifiesWrongBranchAndMissingCompanionBuckets -count=1` proves wrong branch retrieval and missing companion memories can be classified from stable-ID failure-audit rows.
  - Result: future retrieval tuning can start from named failure-audit buckets instead of tuning aggregate recall alone.
```

- [ ] **Step 5: Run final validation**

Run:

```bash
go test ./cmd/goncho-bench -run TestLocomoFailureAuditClassifiesWrongBranchAndMissingCompanionBuckets -count=1
go test . -run TestBenchmarkRoadmapSurfacesLocomoFailureDrivenEvaluationSlice -count=1
make release-metadata-smoke
cd docs-site && npm run build
cd .. && go test ./... -count=1
git diff --check
```

Expected result: every command exits 0.

## Task 5: Commit and push safely

**Files:**
- Stage only files changed by the implemented task.

- [ ] **Step 1: Inspect dirty state**

Run:

```bash
git status --short --branch --untracked-files=all
git diff --stat
```

Expected result: only task-owned files plus the known unrelated `docs/opensource-memory-systems/agentmemory` pointer and `.pi/development-loop*` files are dirty.

- [ ] **Step 2: Stage exact paths only**

For this plan-only slice, stage:

```bash
git add docs/superpowers/plans/2026-05-22-locomo-failure-driven-evaluation.md release_metadata_test.go Makefile TODO.md
```

For a later implementation slice, stage only the files changed by that slice. Never use `git add .`.

- [ ] **Step 3: Verify staged diff and submodule exclusion**

Run:

```bash
git diff --cached --stat
git diff --cached --check
git diff --cached -- docs/opensource-memory-systems/agentmemory
```

Expected result: staged diff has only intended files; the submodule diff command prints nothing.

- [ ] **Step 4: Commit and push**

For this plan-only slice, run:

```bash
git commit -m "docs: plan locomo failure driven evaluation"
git status --short --branch
git push origin main
```

Expected result: commit is created and pushed only if branch status is ahead-only before push.

## Self-review

- Spec coverage: this plan covers failure-audit buckets, wrong branch retrieval, missing companion memories, stable inserted `memory_id` evidence, no-answer-hint benchmark discipline, validation, docs, and safe git delivery.
- Placeholder scan: no placeholder instructions remain; every task names exact files, commands, expected results, and code where code is introduced.
- Scope check: this is one failure-driven evaluation slice. It does not regenerate LOCOMO full-run artifacts, change retrieval behavior, use answer generation, add LLM judges, alter gold data, or tune against LOCOMO gold IDs.
- Stable-ID check: all classification logic reads only memory IDs, ranks, failure reasons, and conversation IDs; it does not inspect answer text or rescue scoring with content-only matching.

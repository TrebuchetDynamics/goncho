# LOCOMO Query Decomposition Recall Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. Use superpowers:executing-plans if implementing inline in the current session.

**Goal:** Add the smallest query-decomposition recall slice that helps multi-part LOCOMO-style questions retrieve each required fact before final ranking.

**Architecture:** Keep the current recall candidate generator as the base path. Add an optional generator wrapper that can split multi-part questions into deterministic subqueries, run the base generator for the original query plus each subquery, then merge and deduplicate by stable `memory_id` before scoring. The wrapper must not inspect answers, gold IDs, or benchmark labels.

**Tech Stack:** Go, existing `RecallEngine` / `recallCandidateGenerator` interfaces, `go test`, stable-ID benchmark discipline.

---

## Scope and guardrails

This plan converts the roadmap recommendation "Add query decomposition so multi-part questions retrieve each required fact before final ranking" into implementation-ready steps. It is not a LOCOMO full-run tuning pass.

Hard constraints:

- Keep LOCOMO scoring centralized and ID-based.
- Preserve stable inserted `memory_id` evidence through every decomposed-query result.
- Use no answer hints, no LLM judges, no answer-text scoring, and no benchmark-specific gold-ID hacks.
- Preserve frozen LOCOMO artifacts until a new date-stamped full run is intentionally generated.
- Do not tune against LOCOMO gold IDs.
- Query decomposition must run before scoring and selection, not after scoring by expected answers.
- Decomposed results must merge and deduplicate by stable `memory_id`.

## File structure

- Create `recall_query_decomposition_test.go`: focused tests for decomposed multi-part recall.
- Create `recall_query_decomposition.go`: optional generator wrapper and deterministic subquery handling.
- Modify `docs/benchmarks/ROADMAP.md` only after behavior is proven, to record the delivered slice.
- Modify `docs-site/src/content/docs/roadmap/benchmark-roadmap.md` only after behavior is proven, to mirror public roadmap status.
- Modify `TODO.md` after validation to record release-state evidence.

## Task 1: Prove the multi-part recall gap with a failing test

**Files:**
- Create: `recall_query_decomposition_test.go`
- Test: `recall_query_decomposition_test.go`

- [ ] **Step 1: Write the failing test**

Create `recall_query_decomposition_test.go` with this test-first content:

```go
package goncho

import (
	"context"
	"slices"
	"testing"
	"time"
)

func TestRecallQueryDecompositionRetrievesEachSubQuestionFact(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	base := queryKeyedRecallGenerator{candidatesByQuery: map[string][]RecallCandidate{
		"Who owns the authentication service and what incident did that owner review?": {
			{
				MemoryID:   "mem-auth-service",
				Content:    "Authentication service handles login and session refresh.",
				ScopeID:    "team",
				CreatedAt:  now,
				Importance: 0.80,
				Provenance: []EvidenceItem{{Kind: "keyword", Score: 0.88, Note: "matched authentication service"}},
			},
		},
		"Who owns the authentication service?": {
			{
				MemoryID:   "mem-auth-owner",
				Content:    "Mira owns the authentication service.",
				ScopeID:    "team",
				CreatedAt:  now,
				Importance: 0.90,
				Provenance: []EvidenceItem{{Kind: "keyword", Score: 0.95, Note: "matched authentication owner"}},
			},
		},
		"What incident did that owner review?": {
			{
				MemoryID:   "mem-auth-incident",
				Content:    "Mira reviewed incident INC-204 for the authentication service.",
				ScopeID:    "team",
				CreatedAt:  now,
				Importance: 0.85,
				Provenance: []EvidenceItem{{Kind: "keyword", Score: 0.93, Note: "matched owner incident"}},
			},
		},
	}}
	engine := newRecallPipelineEngine(
		newQueryDecomposingRecallGenerator(base, fixedRecallSubqueries(
			"Who owns the authentication service?",
			"What incident did that owner review?",
		)),
		recallPipelineOptions{
			pipelineVersion: "query-decomposition-test-v1",
			scoringConfig: RecallScoringConfig{
				Version:       "query-decomposition-test-v1",
				Weights:       map[string]float64{"keyword": 0.85, "scope": 0.15},
				RRFK:          60,
				MMRLambda:     0.70,
				DiversityKeys: []string{"memory_id"},
				TokenBudget:   120,
			},
			now: func() time.Time { return now },
		},
	)

	trace, err := engine.Run(context.Background(), RecallQuery{
		WorkspaceID: "default",
		Peer:        "user-juan",
		Query:       "Who owns the authentication service and what incident did that owner review?",
		ScopeID:     "team",
		Limit:       3,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"mem-auth-service", "mem-auth-owner", "mem-auth-incident"} {
		if !slices.Contains(selectedRecallIDs(trace), want) {
			t.Fatalf("selected IDs = %v, want decomposed fact %q", selectedRecallIDs(trace), want)
		}
	}
}

type queryKeyedRecallGenerator struct {
	candidatesByQuery map[string][]RecallCandidate
}

func (g queryKeyedRecallGenerator) Generate(ctx context.Context, q RecallQuery) ([]RecallCandidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	items := g.candidatesByQuery[q.Query]
	out := make([]RecallCandidate, len(items))
	copy(out, items)
	return out, nil
}
```

- [ ] **Step 2: Run the focused test to verify RED**

Run:

```bash
go test . -run TestRecallQueryDecompositionRetrievesEachSubQuestionFact -count=1
```

Expected result: FAIL because `newQueryDecomposingRecallGenerator` and `fixedRecallSubqueries` do not exist yet.

- [ ] **Step 3: Commit nothing after RED**

Do not commit after RED. Continue to Task 2 in the same slice.

## Task 2: Implement the minimal query-decomposition generator wrapper

**Files:**
- Create: `recall_query_decomposition.go`
- Test: `recall_query_decomposition_test.go`

- [ ] **Step 1: Add the minimal wrapper and fixed-subquery helper**

Create `recall_query_decomposition.go` with this implementation:

```go
package goncho

import (
	"context"
	"strings"
)

type recallSubqueryPlanner func(RecallQuery) []RecallQuery

type queryDecomposingRecallGenerator struct {
	base    recallCandidateGenerator
	planner recallSubqueryPlanner
}

func newQueryDecomposingRecallGenerator(base recallCandidateGenerator, planner recallSubqueryPlanner) recallCandidateGenerator {
	return queryDecomposingRecallGenerator{base: base, planner: planner}
}

func fixedRecallSubqueries(queries ...string) recallSubqueryPlanner {
	return func(q RecallQuery) []RecallQuery {
		out := make([]RecallQuery, 0, len(queries))
		for _, query := range queries {
			query = strings.TrimSpace(query)
			if query == "" || query == q.Query {
				continue
			}
			sub := q
			sub.Query = query
			out = append(out, sub)
		}
		return out
	}
}

func (g queryDecomposingRecallGenerator) Generate(ctx context.Context, q RecallQuery) ([]RecallCandidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	queries := []RecallQuery{q}
	if g.planner != nil {
		queries = append(queries, g.planner(q)...)
	}
	seen := map[string]bool{}
	out := []RecallCandidate{}
	for _, query := range queries {
		if strings.TrimSpace(query.Query) == "" {
			continue
		}
		items, err := g.base.Generate(ctx, query)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if item.MemoryID == "" {
				out = append(out, item)
				continue
			}
			if seen[item.MemoryID] {
				continue
			}
			seen[item.MemoryID] = true
			out = append(out, item)
		}
	}
	return out, nil
}
```

- [ ] **Step 2: Run the focused test to verify GREEN**

Run:

```bash
go test . -run TestRecallQueryDecompositionRetrievesEachSubQuestionFact -count=1
```

Expected result: PASS.

- [ ] **Step 3: Run recall-focused tests**

Run:

```bash
go test . -run 'Test(RecallQueryDecompositionRetrievesEachSubQuestionFact|RecallPipelineCoverageAwareSelectionKeepsGraphCompanion|GraphRecallConnectsOwnerThroughServiceRelation)' -count=1
```

Expected result: PASS.

## Task 3: Add deterministic duplicate handling proof

**Files:**
- Modify: `recall_query_decomposition_test.go`

- [ ] **Step 1: Add the duplicate-memory test**

Append this test to `recall_query_decomposition_test.go`:

```go
func TestRecallQueryDecompositionDeduplicatesStableMemoryIDs(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	base := queryKeyedRecallGenerator{candidatesByQuery: map[string][]RecallCandidate{
		"authentication owner incident": {
			{MemoryID: "mem-auth-owner", Content: "Mira owns authentication.", ScopeID: "team", CreatedAt: now, Importance: 0.80, Provenance: []EvidenceItem{{Kind: "keyword", Score: 0.90}}},
		},
		"authentication owner": {
			{MemoryID: "mem-auth-owner", Content: "Mira owns authentication.", ScopeID: "team", CreatedAt: now, Importance: 0.80, Provenance: []EvidenceItem{{Kind: "keyword", Score: 0.90}}},
		},
		"authentication incident": {
			{MemoryID: "mem-auth-incident", Content: "Mira reviewed INC-204.", ScopeID: "team", CreatedAt: now, Importance: 0.80, Provenance: []EvidenceItem{{Kind: "keyword", Score: 0.90}}},
		},
	}}
	items, err := newQueryDecomposingRecallGenerator(base, fixedRecallSubqueries(
		"authentication owner",
		"authentication incident",
	)).Generate(context.Background(), RecallQuery{Query: "authentication owner incident", ScopeID: "team"})
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.MemoryID)
	}
	if !slices.Equal(ids, []string{"mem-auth-owner", "mem-auth-incident"}) {
		t.Fatalf("merged IDs = %v, want stable memory_id deduplication", ids)
	}
}
```

- [ ] **Step 2: Run the duplicate test**

Run:

```bash
go test . -run TestRecallQueryDecompositionDeduplicatesStableMemoryIDs -count=1
```

Expected result: PASS if Task 2 implemented stable `memory_id` deduplication.

## Task 4: Document the delivered query-decomposition behavior after tests pass

**Files:**
- Modify: `docs/benchmarks/ROADMAP.md`
- Modify: `docs-site/src/content/docs/roadmap/benchmark-roadmap.md`
- Modify: `TODO.md`

- [ ] **Step 1: Add an internal roadmap note**

After tests pass, add this sentence under `LOCOMO implementation gate` in `docs/benchmarks/ROADMAP.md` after the coverage-aware selection note:

```markdown
Query-decomposition recall slice delivered: `TestRecallQueryDecompositionRetrievesEachSubQuestionFact` proves multi-part questions can split into subqueries, retrieve each required stable-ID fact, and merge results before scoring without regenerating LOCOMO full-run artifacts.
```

- [ ] **Step 2: Mirror the public roadmap note**

Add the same sentence under `LOCOMO implementation gate` in `docs-site/src/content/docs/roadmap/benchmark-roadmap.md` after the coverage-aware selection note.

- [ ] **Step 3: Record release-state evidence**

Add this entry near the top of `TODO.md`:

```markdown
- 2026-05-22: LOCOMO query-decomposition recall has its first implementation slice.
  - Evidence target: `go test . -run TestRecallQueryDecompositionRetrievesEachSubQuestionFact -count=1` proves decomposed subqueries can retrieve each required stable-ID fact for a multi-part question before scoring.
  - Result: multi-hop recall work can cover more required facts without answer hints, LLM judges, answer-text scoring, or LOCOMO artifact regeneration.
```

- [ ] **Step 4: Run docs and release checks**

Run:

```bash
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

For the plan-only slice, stage:

```bash
git add docs/superpowers/plans/2026-05-22-locomo-query-decomposition-recall.md release_metadata_test.go Makefile TODO.md
```

For the implementation slice, stage only the files changed by that slice. Never use `git add .`.

- [ ] **Step 3: Verify staged diff and submodule exclusion**

Run:

```bash
git diff --cached --stat
git diff --cached --check
git diff --cached -- docs/opensource-memory-systems/agentmemory
```

Expected result: staged diff has only intended files; the submodule diff command prints nothing.

- [ ] **Step 4: Commit and push**

For the plan-only slice, run:

```bash
git commit -m "docs: plan locomo query decomposition recall"
git status --short --branch
git push origin main
```

Expected result: commit is created and pushed only if branch status is ahead-only before push.

## Self-review

- Spec coverage: this plan covers query decomposition as a LOCOMO improvement lever, a focused failing recall test, stable inserted `memory_id` preservation, deduplication, validation, docs, and safe git delivery.
- Placeholder scan: no placeholder sections remain; every task names exact files, commands, expected results, and code where code is introduced.
- Scope check: this plan is a single query-decomposition recall slice. It does not include LOCOMO full-run regeneration, external adapter changes, graph extraction, answer generation, or LLM judging.
- Stable-ID check: decomposed results merge and deduplicate by stable `memory_id`, and scoring remains ID-based.

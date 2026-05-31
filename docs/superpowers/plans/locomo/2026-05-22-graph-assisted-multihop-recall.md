# Graph-Assisted LOCOMO Multi-Hop Recall Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. Use superpowers:executing-plans if implementing inline in the current session.

**Goal:** Add the smallest graph-assisted recall slice that can improve multi-hop evidence retrieval without changing LOCOMO scoring semantics.

**Architecture:** Keep lexical recall as the base path, then add an optional graph expansion layer that can add evidence-linked companion memories before scoring. Graph-expanded candidates must preserve stable inserted `memory_id` values and expose relation path provenance through `EvidenceItem` entries. The first implementation proves one observable multi-hop recall win, then adds coverage-aware selection only after graph provenance exists.

**Tech Stack:** Go, SQLite-backed Goncho service, existing `RecallEngine` / `recallCandidateGenerator` interfaces, `go test`, stable-ID benchmark discipline.

---

## Scope and guardrails

This plan converts the LOCOMO improvement recommendations into implementation-ready steps. It is not a LOCOMO full-run tuning pass.

Hard constraints:

- Keep LOCOMO scoring centralized and ID-based.
- Preserve stable inserted `memory_id` evidence through every graph-expanded result.
- Use no answer hints, no LLM judges, no answer-text scoring, and no benchmark-specific gold-ID hacks.
- Preserve frozen LOCOMO artifacts until a new date-stamped full run is intentionally generated.
- Do not tune against LOCOMO gold IDs.
- Start with `TestGraphRecallConnectsOwnerThroughServiceRelation` and watch it fail before writing production retrieval code.
- graph-expanded candidates must carry `EvidenceItem{Kind: "graph"` relation path provenance.

## File structure

- Create `recall_graph_multihop_test.go`: focused graph-assisted recall tests.
- Create `recall_graph.go`: graph expansion types and generator wrapper.
- Modify `recall_pipeline.go` only if coverage-aware selection needs a small, tested selector hook.
- Modify `docs/benchmarks/ROADMAP.md` only after behavior is proven, to record the delivered slice.
- Modify `docs-site/src/content/docs/roadmap/benchmark-roadmap.md` only after behavior is proven, to mirror public roadmap status.
- Modify `TODO.md` after validation to record release-state evidence.

## Task 1: Prove the multi-hop recall gap with a failing test

**Files:**
- Create: `recall_graph_multihop_test.go`
- Test: `recall_graph_multihop_test.go`

- [ ] **Step 1: Write the failing test**

Create `recall_graph_multihop_test.go` with this test-first content:

```go
package goncho

import (
	"context"
	"slices"
	"testing"
	"time"
)

func TestGraphRecallConnectsOwnerThroughServiceRelation(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	base := staticRecallGenerator{candidates: []RecallCandidate{
		{
			MemoryID:   "mem-auth-service",
			SourceType: "conclusion",
			Content:    "The authentication service handles login, session refresh, and JWT validation.",
			SessionID:  "sess-auth",
			ScopeID:    "team",
			CreatedAt:  now.Add(-2 * time.Hour),
			Importance: 0.80,
			Provenance: []EvidenceItem{{Kind: "keyword", Score: 0.92, Note: "matched authentication service"}},
		},
	}}
	index := GraphExpansionIndex{
		Memories: map[string]RecallCandidate{
			"mem-auth-owner": {
				MemoryID:   "mem-auth-owner",
				SourceType: "conclusion",
				Content:    "Mira is accountable for component A-17 and reviews production incidents for it.",
				SessionID:  "sess-auth",
				ScopeID:    "team",
				CreatedAt:  now.Add(-90 * time.Minute),
				Importance: 0.85,
			},
		},
		Relations: []GraphRelation{
			{
				FromMemoryID: "mem-auth-service",
				ToMemoryID:   "mem-auth-owner",
				Relation:     "owned_by",
				QueryTerms:   []string{"authentication", "owner"},
				EvidenceID:    "edge-auth-owned-by-mira",
				Score:         0.98,
			},
		},
	}
	engine := newRecallPipelineEngine(
		newGraphExpandingRecallGenerator(base, index),
		recallPipelineOptions{
			pipelineVersion: "graph-test-v1",
			scoringConfig: RecallScoringConfig{
				Version:       "graph-test-v1",
				Weights:       map[string]float64{"keyword": 0.30, "graph": 0.60, "scope": 0.10},
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
		Query:       "Who is the owner for the authentication service?",
		ScopeID:     "team",
		Limit:       2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(selectedRecallIDs(trace), []string{"mem-auth-service", "mem-auth-owner"}) {
		t.Fatalf("selected IDs = %v, want service plus graph-linked owner", selectedRecallIDs(trace))
	}
	owner := trace.Selected[1].Candidate
	if owner.MemoryID != "mem-auth-owner" {
		t.Fatalf("second selected memory = %q, want mem-auth-owner", owner.MemoryID)
	}
	if !candidateHasGraphProvenance(owner, "edge-auth-owned-by-mira") {
		t.Fatalf("owner provenance = %+v, want graph relation path provenance", owner.Provenance)
	}
}

func candidateHasGraphProvenance(candidate RecallCandidate, evidenceID string) bool {
	for _, item := range candidate.Provenance {
		if item.Kind == "graph" && item.ID == evidenceID {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run the focused test to verify RED**

Run:

```bash
go test . -run TestGraphRecallConnectsOwnerThroughServiceRelation -count=1
```

Expected result: FAIL because `GraphExpansionIndex`, `GraphRelation`, and `newGraphExpandingRecallGenerator` do not exist yet.

- [ ] **Step 3: Commit nothing yet**

Do not commit after RED. Continue to Task 2 in the same slice.

## Task 2: Implement the minimal graph expansion wrapper

**Files:**
- Create: `recall_graph.go`
- Test: `recall_graph_multihop_test.go`

- [ ] **Step 1: Add minimal graph expansion types and generator**

Create `recall_graph.go` with this minimal implementation:

```go
package goncho

import (
	"context"
	"strings"
)

type GraphExpansionIndex struct {
	Memories  map[string]RecallCandidate
	Relations []GraphRelation
}

type GraphRelation struct {
	FromMemoryID string
	ToMemoryID   string
	Relation     string
	QueryTerms   []string
	EvidenceID   string
	Score        float64
}

type graphExpandingRecallGenerator struct {
	base  recallCandidateGenerator
	index GraphExpansionIndex
}

func newGraphExpandingRecallGenerator(base recallCandidateGenerator, index GraphExpansionIndex) recallCandidateGenerator {
	return graphExpandingRecallGenerator{base: base, index: index}
}

func (g graphExpandingRecallGenerator) Generate(ctx context.Context, q RecallQuery) ([]RecallCandidate, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	base, err := g.base.Generate(ctx, q)
	if err != nil {
		return nil, err
	}
	out := make([]RecallCandidate, len(base))
	copy(out, base)
	seen := make(map[string]bool, len(out))
	for _, candidate := range out {
		seen[candidate.MemoryID] = true
	}
	for _, relation := range g.index.Relations {
		if !seen[relation.FromMemoryID] || seen[relation.ToMemoryID] || !graphRelationMatchesQuery(q.Query, relation.QueryTerms) {
			continue
		}
		target, ok := g.index.Memories[relation.ToMemoryID]
		if !ok || recallScopeMismatch(q, target) {
			continue
		}
		target.Provenance = append(cloneEvidenceItems(target.Provenance), EvidenceItem{
			Kind:   "graph",
			ID:     relation.EvidenceID,
			Source: relation.FromMemoryID,
			Note:   relation.FromMemoryID + " -> " + relation.Relation + " -> " + relation.ToMemoryID,
			Score:  relation.Score,
		})
		out = append(out, target)
		seen[target.MemoryID] = true
	}
	return out, nil
}

func graphRelationMatchesQuery(query string, terms []string) bool {
	query = strings.ToLower(query)
	for _, term := range terms {
		if !strings.Contains(query, strings.ToLower(strings.TrimSpace(term))) {
			return false
		}
	}
	return true
}

func cloneEvidenceItems(items []EvidenceItem) []EvidenceItem {
	out := make([]EvidenceItem, len(items))
	copy(out, items)
	return out
}
```

- [ ] **Step 2: Run the focused test to verify GREEN**

Run:

```bash
go test . -run TestGraphRecallConnectsOwnerThroughServiceRelation -count=1
```

Expected result: PASS.

- [ ] **Step 3: Run package tests around recall**

Run:

```bash
go test . -run 'Test(Recall|GraphRecall)' -count=1
```

Expected result: PASS.

## Task 3: Preserve relation path provenance in traces

**Files:**
- Modify: `recall_graph_multihop_test.go`
- Modify: `recall_graph.go`

- [ ] **Step 1: Strengthen the test for relation path provenance**

Add this assertion inside `TestGraphRecallConnectsOwnerThroughServiceRelation` after the owner candidate is read:

```go
if !candidateHasGraphNote(owner, "mem-auth-service -> owned_by -> mem-auth-owner") {
	t.Fatalf("owner provenance = %+v, want relation path provenance", owner.Provenance)
}
```

Add this helper below `candidateHasGraphProvenance`:

```go
func candidateHasGraphNote(candidate RecallCandidate, note string) bool {
	for _, item := range candidate.Provenance {
		if item.Kind == "graph" && item.Note == note {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run the focused test to verify it still fails if provenance is absent**

Run:

```bash
go test . -run TestGraphRecallConnectsOwnerThroughServiceRelation -count=1
```

Expected result: PASS if Task 2 already wrote the relation path note exactly; FAIL if the note is missing or unstable. If it fails, update only the graph evidence note string in `recall_graph.go`.

- [ ] **Step 3: Confirm stable JSON can carry graph provenance**

Run:

```bash
go test . -run TestRecallTraceStableIDAndJSONFixture -count=1
```

Expected result: PASS. This proves existing trace serialization still accepts graph provenance.

## Task 4: Add coverage-aware selection only after graph recall works

**Files:**
- Modify: `recall_pipeline_test.go`
- Modify: `recall_pipeline.go`

- [ ] **Step 1: Write a failing coverage-aware selection test**

Add this test to `recall_pipeline_test.go`:

```go
func TestRecallPipelineCoverageAwareSelectionKeepsGraphCompanion(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	engine := newRecallPipelineEngine(staticRecallGenerator{candidates: []RecallCandidate{
		{MemoryID: "mem-auth-service", Content: "Authentication service handles login flows.", ScopeID: "team", CreatedAt: now, Importance: 0.8, Provenance: []EvidenceItem{{Kind: "keyword", Score: 1.0}}},
		{MemoryID: "mem-auth-service-dup", Content: "Authentication service handles login flows and session refresh.", ScopeID: "team", CreatedAt: now, Importance: 0.8, Provenance: []EvidenceItem{{Kind: "keyword", Score: 0.99}}},
		{MemoryID: "mem-auth-owner", Content: "Mira owns component A-17.", ScopeID: "team", CreatedAt: now, Importance: 0.8, Provenance: []EvidenceItem{{Kind: "graph", Score: 0.98, Note: "mem-auth-service -> owned_by -> mem-auth-owner"}}},
	}}, recallPipelineOptions{
		pipelineVersion: "coverage-test-v1",
		scoringConfig: RecallScoringConfig{Version: "coverage-test-v1", Weights: map[string]float64{"keyword": 0.45, "graph": 0.45, "scope": 0.10}, RRFK: 60, MMRLambda: 0.70, DiversityKeys: []string{"memory_id"}, TokenBudget: 120},
		now: func() time.Time { return now },
	})
	trace, err := engine.Run(context.Background(), RecallQuery{WorkspaceID: "default", Peer: "user-juan", Query: "authentication owner", ScopeID: "team", Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(selectedRecallIDs(trace), []string{"mem-auth-service", "mem-auth-owner"}) {
		t.Fatalf("selected IDs = %v, want coverage-aware selection", selectedRecallIDs(trace))
	}
}
```

- [ ] **Step 2: Run coverage test to verify RED**

Run:

```bash
go test . -run TestRecallPipelineCoverageAwareSelectionKeepsGraphCompanion -count=1
```

Expected result: FAIL until selection prefers complementary graph evidence over near-duplicate lexical evidence.

- [ ] **Step 3: Implement the smallest coverage-aware selection change**

In `recall_pipeline.go`, adjust `selectCandidates` so candidates with graph provenance are not displaced by near-duplicate lexical candidates when the limit is small. Keep the change local to selection and avoid changing candidate generation.

Use this policy:

```go
// When two candidates have similar lexical content and one selected candidate already
// covers the same lexical branch, prefer a graph-provenance candidate that adds a
// distinct stable memory_id relation path.
```

- [ ] **Step 4: Run focused and package tests**

Run:

```bash
go test . -run 'Test(RecallPipelineCoverageAwareSelectionKeepsGraphCompanion|GraphRecallConnectsOwnerThroughServiceRelation)' -count=1
go test ./... -count=1
```

Expected result: both commands PASS.

## Task 5: Document the delivered retrieval behavior after tests pass

**Files:**
- Modify: `docs/benchmarks/ROADMAP.md`
- Modify: `docs-site/src/content/docs/roadmap/benchmark-roadmap.md`
- Modify: `TODO.md`

- [ ] **Step 1: Add an internal roadmap note**

Add this sentence under `LOCOMO implementation gate` in `docs/benchmarks/ROADMAP.md` after tests prove behavior:

```markdown
First graph-assisted implementation slice delivered: `TestGraphRecallConnectsOwnerThroughServiceRelation` proves graph-expanded multi-hop recall can retrieve a stable-ID companion memory with relation path provenance before any LOCOMO full-run artifact is regenerated.
```

- [ ] **Step 2: Mirror the public roadmap note**

Add the same sentence under `LOCOMO implementation gate` in `docs-site/src/content/docs/roadmap/benchmark-roadmap.md`.

- [ ] **Step 3: Record release-state evidence**

Add this entry near the top of `TODO.md`:

```markdown
- 2026-05-22: Graph-assisted LOCOMO multi-hop recall has its first implementation slice.
  - Evidence target: `go test . -run TestGraphRecallConnectsOwnerThroughServiceRelation -count=1` proves graph-expanded recall retrieves a stable-ID companion memory with relation path provenance.
  - Result: LOCOMO improvement work can move from recommendations to a measured graph-assisted recall slice without changing frozen benchmark artifacts or scoring by answer text.
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

## Task 6: Commit and push safely

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
git add docs/superpowers/plans/2026-05-22-locomo-graph-assisted-multihop-recall.md release_metadata_test.go Makefile TODO.md
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

Run:

```bash
git commit -m "docs: plan locomo graph assisted recall"
git status --short --branch
git push origin main
```

Expected result: commit is created and pushed only if branch status is ahead-only before push.

## Self-review

- Spec coverage: this plan covers the recommendation-to-implementation boundary, the graph-assisted multi-hop recall test, stable inserted `memory_id` preservation, relation path provenance, coverage-aware selection, validation, and safe git delivery.
- Placeholder scan: no placeholder sections remain; every task names exact files, commands, expected results, and code where code is introduced.
- Scope check: this plan is a single graph-assisted recall slice. It does not include LOCOMO full-run regeneration, external adapter changes, answer generation, or LLM judging.
- Stable-ID check: every graph-expanded result keeps the original memory ID, and scoring remains ID-based.

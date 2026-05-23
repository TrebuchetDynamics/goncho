# LOCOMO Temporal and Speaker Routing Recall Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. Use superpowers:executing-plans if implementing inline in the current session.

**Goal:** Add the smallest temporal and speaker-routing recall slice that ranks current truth above superseded evidence and keeps who-said-what in the right conversation branch.

**Architecture:** Keep the existing candidate generator and stable-ID scoring harness. Add a small selection/scoring hook that reads conservative provenance metadata already carried by `EvidenceItem`: temporal evidence marks current or superseded facts, and speaker evidence marks who asserted a fact. The hook adjusts ranking and emits warnings; it never deletes historical evidence and never uses answers, gold IDs, or benchmark labels.

**Tech Stack:** Go, existing `RecallEngine` / `recallCandidateGenerator` interfaces, `RecallWarning`, `EvidenceItem`, `go test`, stable-ID benchmark discipline.

---

## Scope and guardrails

This plan converts the roadmap recommendation "Improve temporal and speaker routing so changed facts, chronology, and who-said-what are ranked in the right conversation branch" into implementation-ready steps. It is not a LOCOMO full-run tuning pass.

Hard constraints:

- Keep LOCOMO scoring centralized and ID-based.
- Preserve stable inserted `memory_id` evidence through every temporal or speaker-routed result.
- Superseded evidence remains preserved in `trace.Candidates`; it is not destructively deleted.
- The required audit phrase is: superseded evidence remains preserved.
- Use no answer hints, no LLM judges, no answer-text scoring, and no benchmark-specific gold-ID hacks.
- Preserve frozen LOCOMO artifacts until a new date-stamped full run is intentionally generated.
- Do not tune against LOCOMO gold IDs.
- Start with focused failing recall tests before production retrieval changes.
- Current-truth routing must expose a warning when superseded evidence is present in the candidate pool.
- Speaker routing must only use explicit speaker provenance or candidate metadata, not inferred answer text.

## File structure

- Create `recall_temporal_speaker_test.go`: focused recall tests for current-truth and who-said-what routing.
- Modify `recall_ir.go`: add one warning code for superseded evidence observed during recall.
- Modify `recall_pipeline.go`: add a small temporal/speaker routing adjustment local to scoring/selection.
- Modify `docs/benchmarks/ROADMAP.md` only after behavior is proven, to record the delivered slice.
- Modify `docs-site/src/content/docs/roadmap/benchmark-roadmap.md` only after behavior is proven, to mirror public roadmap status.
- Modify `TODO.md` after validation to record release-state evidence.

## Task 1: Prove temporal current-truth routing with a failing test

**Files:**
- Create: `recall_temporal_speaker_test.go`
- Test: `recall_temporal_speaker_test.go`

- [ ] **Step 1: Write the failing temporal routing test**

Create `recall_temporal_speaker_test.go` with this test-first content:

```go
package goncho

import (
	"context"
	"slices"
	"testing"
	"time"
)

func TestRecallTemporalRoutingPrefersCurrentFactAndWarnsOnSupersededEvidence(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	engine := newRecallPipelineEngine(staticRecallGenerator{candidates: []RecallCandidate{
		{
			MemoryID:   "mem-owner-old",
			Content:    "Mira owns component A-17.",
			ScopeID:    "team",
			CreatedAt:  now.Add(-48 * time.Hour),
			Importance: 0.95,
			Provenance: []EvidenceItem{
				{Kind: "keyword", Score: 1.00, Note: "matched component owner"},
				{Kind: "temporal", Score: 0.10, Note: "superseded_by=mem-owner-current"},
			},
		},
		{
			MemoryID:   "mem-owner-current",
			Content:    "Nadia now owns component A-17.",
			ScopeID:    "team",
			CreatedAt:  now.Add(-2 * time.Hour),
			Importance: 0.70,
			Provenance: []EvidenceItem{
				{Kind: "keyword", Score: 0.86, Note: "matched component owner"},
				{Kind: "temporal", Score: 1.00, Note: "current_fact"},
			},
		},
	}}, recallPipelineOptions{
		pipelineVersion: "temporal-routing-test-v1",
		scoringConfig: RecallScoringConfig{
			Version:       "temporal-routing-test-v1",
			Weights:       map[string]float64{"keyword": 0.65, "recency": 0.10, "importance": 0.15, "scope": 0.10},
			RRFK:          60,
			MMRLambda:     0.70,
			DiversityKeys: []string{"memory_id"},
			TokenBudget:   120,
		},
		now: func() time.Time { return now },
	})

	trace, err := engine.Run(context.Background(), RecallQuery{
		WorkspaceID: "default",
		Peer:        "user-juan",
		Query:       "Who owns component A-17 now?",
		ScopeID:     "team",
		Limit:       1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(selectedRecallIDs(trace), []string{"mem-owner-current"}) {
		t.Fatalf("selected IDs = %v, want current owner", selectedRecallIDs(trace))
	}
	if !traceHasWarning(trace, RecallWarningSupersededEvidenceObserved) {
		t.Fatalf("warnings = %+v, want superseded-evidence warning", trace.Warnings)
	}
	if !candidateIDSeen(trace.Candidates, "mem-owner-old") {
		t.Fatalf("candidates = %+v, want superseded evidence preserved", trace.Candidates)
	}
}

func candidateIDSeen(items []ScoredRecallCandidate, memoryID string) bool {
	for _, item := range items {
		if item.Candidate.MemoryID == memoryID {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run the focused temporal test to verify RED**

Run:

```bash
go test . -run TestRecallTemporalRoutingPrefersCurrentFactAndWarnsOnSupersededEvidence -count=1
```

Expected result: FAIL because `RecallWarningSupersededEvidenceObserved` does not exist yet and because the current fact is not preferred by temporal routing.

- [ ] **Step 3: Commit nothing after RED**

Do not commit after RED. Continue to Task 2 in the same slice.

## Task 2: Add the minimal temporal routing warning and score adjustment

**Files:**
- Modify: `recall_ir.go`
- Modify: `recall_pipeline.go`
- Test: `recall_temporal_speaker_test.go`

- [ ] **Step 1: Add the warning code**

In `recall_ir.go`, add this constant after `RecallWarningTokenBudgetTruncated`:

```go
	RecallWarningSupersededEvidenceObserved = "superseded_evidence_observed"
```

- [ ] **Step 2: Add temporal routing helpers**

In `recall_pipeline.go`, add these helpers near `recallCoverageBonus`:

```go
const recallTemporalCurrentBonus = 0.08
const recallTemporalSupersededPenalty = 0.20

func recallTemporalAdjustment(candidate ScoredRecallCandidate, query string) float64 {
	if !recallQueryAsksCurrentTruth(query) {
		return 0
	}
	for _, evidence := range candidate.Candidate.Provenance {
		if evidence.Kind != "temporal" {
			continue
		}
		note := strings.ToLower(strings.TrimSpace(evidence.Note))
		if strings.Contains(note, "superseded_by=") || strings.Contains(note, "superseded") {
			return -recallTemporalSupersededPenalty
		}
		if strings.Contains(note, "current_fact") || strings.Contains(note, "valid_now") {
			return recallTemporalCurrentBonus
		}
	}
	return 0
}

func recallQueryAsksCurrentTruth(query string) bool {
	query = strings.ToLower(query)
	for _, marker := range []string{" now", "current", "currently", "latest", "today"} {
		if strings.Contains(query, marker) {
			return true
		}
	}
	return false
}

func recallHasSupersededEvidence(candidates []ScoredRecallCandidate) bool {
	for _, candidate := range candidates {
		for _, evidence := range candidate.Candidate.Provenance {
			if evidence.Kind != "temporal" {
				continue
			}
			note := strings.ToLower(strings.TrimSpace(evidence.Note))
			if strings.Contains(note, "superseded_by=") || strings.Contains(note, "superseded") {
				return true
			}
		}
	}
	return false
}
```

- [ ] **Step 3: Apply the temporal adjustment in selection**

In `selectCandidates`, update the effective score calculation inside the `remaining` loop:

```go
penalty := recallDiversityPenalty(remaining[i], selected, e.opts.scoringConfig)
coverageBonus := recallCoverageBonus(remaining[i], selected)
temporalAdjustment := recallTemporalAdjustment(remaining[i], q.Query)
effectiveScore := remaining[i].Score.FinalScore - penalty + coverageBonus + temporalAdjustment
```

After choosing `chosen`, add the same adjustment to the final selected score and `WhySelected`:

```go
temporalAdjustment := recallTemporalAdjustment(chosen, q.Query)
chosen.Score.FinalScore = roundRecallFloat(chosen.Score.FinalScore - chosen.Score.DiversityPenalty + coverageBonus + temporalAdjustment)
if temporalAdjustment != 0 {
	chosen.Score.WhySelected = append(chosen.Score.WhySelected, fmt.Sprintf("temporal_adjustment=%.6f", temporalAdjustment))
}
```

After the scope-exclusion warning block, append the superseded warning when applicable:

```go
if recallQueryAsksCurrentTruth(q.Query) && recallHasSupersededEvidence(eligible) {
	warnings = appendRecallWarnings(warnings, RecallWarning{
		Code:     RecallWarningSupersededEvidenceObserved,
		Stage:    RecallStageSelect,
		Severity: RecallWarningInfo,
		Message:  "recall candidates include superseded evidence; current-truth routing adjusted selection",
	})
}
```

- [ ] **Step 4: Run the focused temporal test to verify GREEN**

Run:

```bash
go test . -run TestRecallTemporalRoutingPrefersCurrentFactAndWarnsOnSupersededEvidence -count=1
```

Expected result: PASS.

## Task 3: Prove speaker branch routing with a failing test

**Files:**
- Modify: `recall_temporal_speaker_test.go`
- Modify: `recall_pipeline.go`

- [ ] **Step 1: Add the speaker routing test**

Append this test to `recall_temporal_speaker_test.go`:

```go
func TestRecallSpeakerRoutingKeepsWhoSaidWhatInBranch(t *testing.T) {
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	engine := newRecallPipelineEngine(staticRecallGenerator{candidates: []RecallCandidate{
		{
			MemoryID:   "mem-juan-theme",
			Content:    "Juan said he prefers dark theme for dense dashboards.",
			AgentID:    "juan",
			ScopeID:    "team",
			CreatedAt:  now.Add(-30 * time.Minute),
			Importance: 0.95,
			Provenance: []EvidenceItem{{Kind: "keyword", Score: 1.00}, {Kind: "speaker", Source: "juan", Score: 1.00, Note: "speaker=juan"}},
		},
		{
			MemoryID:   "mem-mira-theme",
			Content:    "Mira said Juan prefers light theme during demos.",
			AgentID:    "mira",
			ScopeID:    "team",
			CreatedAt:  now.Add(-2 * time.Hour),
			Importance: 0.70,
			Provenance: []EvidenceItem{{Kind: "keyword", Score: 0.92}, {Kind: "speaker", Source: "mira", Score: 1.00, Note: "speaker=mira"}},
		},
	}}, recallPipelineOptions{
		pipelineVersion: "speaker-routing-test-v1",
		scoringConfig: RecallScoringConfig{
			Version:       "speaker-routing-test-v1",
			Weights:       map[string]float64{"keyword": 0.75, "recency": 0.10, "importance": 0.10, "scope": 0.05},
			RRFK:          60,
			MMRLambda:     0.70,
			DiversityKeys: []string{"memory_id"},
			TokenBudget:   120,
		},
		now: func() time.Time { return now },
	})

	trace, err := engine.Run(context.Background(), RecallQuery{
		WorkspaceID: "default",
		Peer:        "user-juan",
		Query:       "What did Mira say Juan preferred for demos?",
		ScopeID:     "team",
		Limit:       1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(selectedRecallIDs(trace), []string{"mem-mira-theme"}) {
		t.Fatalf("selected IDs = %v, want Mira speaker branch", selectedRecallIDs(trace))
	}
}
```

- [ ] **Step 2: Run the speaker test to verify RED**

Run:

```bash
go test . -run TestRecallSpeakerRoutingKeepsWhoSaidWhatInBranch -count=1
```

Expected result: FAIL because speaker routing does not yet affect selection.

- [ ] **Step 3: Add minimal speaker routing helpers**

In `recall_pipeline.go`, add these helpers near the temporal helpers:

```go
const recallSpeakerMatchBonus = 0.08

func recallSpeakerAdjustment(candidate ScoredRecallCandidate, query string) float64 {
	query = strings.ToLower(query)
	if strings.TrimSpace(query) == "" {
		return 0
	}
	for _, evidence := range candidate.Candidate.Provenance {
		if evidence.Kind != "speaker" {
			continue
		}
		speaker := strings.ToLower(strings.TrimSpace(evidence.Source))
		if speaker == "" {
			speaker = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(evidence.Note)), "speaker=")
		}
		if speaker != "" && strings.Contains(query, speaker) {
			return recallSpeakerMatchBonus
		}
	}
	if candidate.Candidate.AgentID != "" && strings.Contains(query, strings.ToLower(candidate.Candidate.AgentID)) {
		return recallSpeakerMatchBonus
	}
	return 0
}
```

- [ ] **Step 4: Apply the speaker adjustment in selection**

Add `speakerAdjustment := recallSpeakerAdjustment(..., q.Query)` beside `temporalAdjustment` in both effective-score and chosen-score calculations:

```go
speakerAdjustment := recallSpeakerAdjustment(remaining[i], q.Query)
effectiveScore := remaining[i].Score.FinalScore - penalty + coverageBonus + temporalAdjustment + speakerAdjustment
```

For the chosen item:

```go
speakerAdjustment := recallSpeakerAdjustment(chosen, q.Query)
chosen.Score.FinalScore = roundRecallFloat(chosen.Score.FinalScore - chosen.Score.DiversityPenalty + coverageBonus + temporalAdjustment + speakerAdjustment)
if speakerAdjustment > 0 {
	chosen.Score.WhySelected = append(chosen.Score.WhySelected, fmt.Sprintf("speaker_adjustment=%.6f", speakerAdjustment))
}
```

- [ ] **Step 5: Run the speaker test to verify GREEN**

Run:

```bash
go test . -run TestRecallSpeakerRoutingKeepsWhoSaidWhatInBranch -count=1
```

Expected result: PASS.

## Task 4: Document the delivered temporal/speaker routing behavior after tests pass

**Files:**
- Modify: `docs/benchmarks/ROADMAP.md`
- Modify: `docs-site/src/content/docs/roadmap/benchmark-roadmap.md`
- Modify: `TODO.md`

- [ ] **Step 1: Add an internal roadmap note**

After tests pass, add this sentence under `LOCOMO implementation gate` in `docs/benchmarks/ROADMAP.md` after the query-decomposition note:

```markdown
Temporal and speaker routing slice delivered: `TestRecallTemporalRoutingPrefersCurrentFactAndWarnsOnSupersededEvidence` and `TestRecallSpeakerRoutingKeepsWhoSaidWhatInBranch` prove current facts and who-said-what branches can influence recall selection while preserving superseded evidence and stable memory IDs.
```

- [ ] **Step 2: Mirror the public roadmap note**

Add the same sentence under `LOCOMO implementation gate` in `docs-site/src/content/docs/roadmap/benchmark-roadmap.md` after the query-decomposition note.

- [ ] **Step 3: Record release-state evidence**

Add this entry near the top of `TODO.md`:

```markdown
- 2026-05-22: LOCOMO temporal and speaker routing has its first implementation slice.
  - Evidence target: `go test . -run 'TestRecall(TemporalRoutingPrefersCurrentFactAndWarnsOnSupersededEvidence|SpeakerRoutingKeepsWhoSaidWhatInBranch)' -count=1` proves current-truth and who-said-what routing can affect recall selection while preserving superseded evidence.
  - Result: future temporal LOCOMO work can distinguish current truth from past truth without answer hints, LLM judges, answer-text scoring, or LOCOMO artifact regeneration.
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
git add docs/superpowers/plans/2026-05-22-locomo-temporal-speaker-routing-recall.md release_metadata_test.go Makefile TODO.md
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
git commit -m "docs: plan locomo temporal speaker routing recall"
git status --short --branch
git push origin main
```

For the implementation slice, use:

```bash
git commit -m "feat: add temporal speaker recall routing"
git status --short --branch
git push origin main
```

Expected result: commit is created and pushed only if branch status is ahead-only before push.

## Self-review

- Spec coverage: this plan covers temporal and speaker routing as a LOCOMO improvement lever, focused failing recall tests, current-truth selection, superseded-evidence warning, who-said-what branch routing, stable inserted `memory_id` preservation, validation, docs, and safe git delivery.
- Placeholder scan: no placeholder sections remain; every task names exact files, commands, expected results, and code where code is introduced.
- Scope check: this plan is a single temporal/speaker recall slice. It does not include LOCOMO full-run regeneration, external adapter changes, graph extraction, query decomposition changes, answer generation, or LLM judging.
- Stable-ID check: temporal and speaker adjustments affect ranking only; scoring remains centered on stable inserted `memory_id` evidence and historical superseded evidence stays visible.

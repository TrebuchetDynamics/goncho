# Skill Learning Governance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a local SQLite-backed Goncho API for reviewable skill-learning proposals so generated skills remain auditable and untrusted until reviewed.

**Architecture:** Follow Goncho's existing `memory_proposals.go` and `review.go` style: a focused root-package file owns types, validation, table creation, CRUD, and review transitions. `RunMigrations` includes the table DDL so embedded Goncho callers get the audit trail by default.

**Tech Stack:** Go 1.25, `database/sql`, SQLite via existing test driver, Goncho root package tests.

---

## File Structure

- Create: `/home/xel/git/sages-openclaw/workspace-mineru/goncho/skill_learning_proposals.go`
  - Owns `SkillLearningProposal`, params, statuses, validation, table DDL, submit/list/get/approve/reject functions, and service methods.
- Create: `/home/xel/git/sages-openclaw/workspace-mineru/goncho/skill_learning_proposals_test.go`
  - Focused tests for persistence, invalid input, scoped pending lists, review transitions, duplicate review failures, and evidence round-trip.
- Modify: `/home/xel/git/sages-openclaw/workspace-mineru/goncho/migrations.go`
  - Append `gonchoSkillLearningProposalDDL` to `RunMigrations`.

## Task 1: Add failing tests for skill-learning proposals

**Files:**
- Create: `/home/xel/git/sages-openclaw/workspace-mineru/goncho/skill_learning_proposals_test.go`

- [ ] **Step 1: Write the failing test file**

Create the test file with these tests:

```go
package goncho

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestSkillLearningProposal_SubmitAndEvidenceRoundTrip(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	createdAt := time.Date(2026, 5, 20, 15, 0, 0, 0, time.UTC)
	ref, err := svc.SubmitSkillLearningProposal(ctx, SkillLearningProposalCreateParams{
		WorkspaceID: svc.workspaceID,
		SkillName:   "gormes-git",
		SourceTask:  "resolved repeated git blocker with build artifact evidence",
		DraftBody:   "# gormes-git\n\nCheck build artifacts before commit.",
		CreatedBy:   "agent:mineru",
		Evidence: map[string]any{
			"tool_calls": float64(7),
			"validation": "git diff --check",
			"outcome":    "success",
		},
		CreatedAt: createdAt,
	})
	if err != nil {
		t.Fatalf("SubmitSkillLearningProposal: %v", err)
	}
	if ref.ProposalID == "" || ref.Status != SkillLearningProposalPending {
		t.Fatalf("ref = %+v, want pending id", ref)
	}

	proposal, err := svc.GetSkillLearningProposal(ctx, ref.ProposalID)
	if err != nil {
		t.Fatalf("GetSkillLearningProposal: %v", err)
	}
	if proposal.SkillName != "gormes-git" || proposal.Status != SkillLearningProposalPending {
		t.Fatalf("proposal = %+v", proposal)
	}
	var evidence map[string]any
	if err := json.Unmarshal(proposal.EvidenceJSON, &evidence); err != nil {
		t.Fatalf("evidence json: %v", err)
	}
	if evidence["validation"] != "git diff --check" || evidence["outcome"] != "success" {
		t.Fatalf("evidence = %#v", evidence)
	}
}

func TestSkillLearningProposal_InvalidInputFailsClosed(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	cases := []SkillLearningProposalCreateParams{
		{SkillName: "", SourceTask: "task", DraftBody: "draft", CreatedBy: "agent:mineru"},
		{SkillName: "skill", SourceTask: "", DraftBody: "draft", CreatedBy: "agent:mineru"},
		{SkillName: "skill", SourceTask: "task", DraftBody: "", CreatedBy: "agent:mineru"},
		{SkillName: "skill", SourceTask: "task", DraftBody: "draft", CreatedBy: ""},
	}
	for _, tc := range cases {
		if _, err := svc.SubmitSkillLearningProposal(ctx, tc); err == nil {
			t.Fatalf("expected invalid proposal to fail: %+v", tc)
		}
	}
}

func TestSkillLearningProposal_ListPendingScopedAndNewestFirst(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()
	base := time.Date(2026, 5, 20, 16, 0, 0, 0, time.UTC)

	_, _ = svc.SubmitSkillLearningProposal(ctx, SkillLearningProposalCreateParams{WorkspaceID: svc.workspaceID, SkillName: "old", SourceTask: "task old", DraftBody: "draft old", CreatedBy: "agent:mineru", CreatedAt: base})
	newer, _ := svc.SubmitSkillLearningProposal(ctx, SkillLearningProposalCreateParams{WorkspaceID: svc.workspaceID, SkillName: "new", SourceTask: "task new", DraftBody: "draft new", CreatedBy: "agent:mineru", CreatedAt: base.Add(time.Minute)})
	_, _ = svc.SubmitSkillLearningProposal(ctx, SkillLearningProposalCreateParams{WorkspaceID: "other-workspace", SkillName: "other", SourceTask: "task other", DraftBody: "draft other", CreatedBy: "agent:mineru", CreatedAt: base.Add(2 * time.Minute)})

	items, err := svc.ListPendingSkillLearningProposals(ctx, SkillLearningProposalQuery{WorkspaceID: svc.workspaceID, Limit: 10})
	if err != nil {
		t.Fatalf("ListPendingSkillLearningProposals: %v", err)
	}
	if len(items.Items) != 2 {
		t.Fatalf("pending count = %d, want 2: %+v", len(items.Items), items.Items)
	}
	if items.Items[0].ProposalID != newer.ProposalID || items.Items[0].SkillName != "new" {
		t.Fatalf("order = %+v, want newest first", items.Items)
	}
}

func TestSkillLearningProposal_ApproveRejectAndDuplicateReview(t *testing.T) {
	svc, cleanup := newTestService(t)
	defer cleanup()
	ctx := context.Background()

	approveRef, _ := svc.SubmitSkillLearningProposal(ctx, SkillLearningProposalCreateParams{SkillName: "approve-me", SourceTask: "task", DraftBody: "draft", CreatedBy: "agent:mineru"})
	approved, err := svc.ApproveSkillLearningProposal(ctx, SkillLearningProposalReviewParams{ProposalID: approveRef.ProposalID, ReviewedBy: "human:juan", ReviewReason: "validated against tests"})
	if err != nil {
		t.Fatalf("ApproveSkillLearningProposal: %v", err)
	}
	if approved.Status != SkillLearningProposalApproved || approved.ReviewedBy != "human:juan" || approved.ReviewReason != "validated against tests" || approved.ReviewedAt == nil {
		t.Fatalf("approved = %+v", approved)
	}
	if _, err := svc.RejectSkillLearningProposal(ctx, SkillLearningProposalReviewParams{ProposalID: approveRef.ProposalID, ReviewedBy: "human:juan", ReviewReason: "second review"}); err == nil {
		t.Fatal("expected duplicate review to fail")
	}

	rejectRef, _ := svc.SubmitSkillLearningProposal(ctx, SkillLearningProposalCreateParams{SkillName: "reject-me", SourceTask: "task", DraftBody: "draft", CreatedBy: "agent:mineru"})
	rejected, err := svc.RejectSkillLearningProposal(ctx, SkillLearningProposalReviewParams{ProposalID: rejectRef.ProposalID, ReviewedBy: "human:juan", ReviewReason: "overfit to one invoice"})
	if err != nil {
		t.Fatalf("RejectSkillLearningProposal: %v", err)
	}
	if rejected.Status != SkillLearningProposalRejected || rejected.ReviewReason != "overfit to one invoice" {
		t.Fatalf("rejected = %+v", rejected)
	}
}
```

- [ ] **Step 2: Run tests to verify RED**

Run:

```bash
cd /home/xel/git/sages-openclaw/workspace-mineru/goncho
go test ./... -run 'TestSkillLearningProposal' -count=1
```

Expected: FAIL because `SkillLearningProposalCreateParams` and related methods are undefined.

## Task 2: Implement the proposal API and migration DDL

**Files:**
- Create: `/home/xel/git/sages-openclaw/workspace-mineru/goncho/skill_learning_proposals.go`
- Modify: `/home/xel/git/sages-openclaw/workspace-mineru/goncho/migrations.go`

- [ ] **Step 1: Add implementation file**

Create `skill_learning_proposals.go` with types and functions matching the tests. Use `ensureSkillLearningProposalTable(ctx, db)` inside public functions so existing callers work even before `RunMigrations`.

- [ ] **Step 2: Add migration DDL to `RunMigrations`**

Change the loop in `migrations.go` from:

```go
for _, stmt := range append(gonchoObservationDDL, gonchoReviewDDL...) {
```

to:

```go
for _, stmt := range append(append(gonchoObservationDDL, gonchoReviewDDL...), gonchoSkillLearningProposalDDL...) {
```

- [ ] **Step 3: Run focused tests to verify GREEN**

Run:

```bash
cd /home/xel/git/sages-openclaw/workspace-mineru/goncho
go test ./... -run 'TestSkillLearningProposal' -count=1
```

Expected: PASS for all `TestSkillLearningProposal*` tests.

## Task 3: Full validation and commit

**Files:**
- Validate all modified files.

- [ ] **Step 1: Run full Go suite**

Run:

```bash
cd /home/xel/git/sages-openclaw/workspace-mineru/goncho
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 2: Check formatting and git diff**

Run:

```bash
gofmt -w skill_learning_proposals.go skill_learning_proposals_test.go migrations.go
git diff --check
git status --short
```

Expected: no `git diff --check` output; status shows only intended files.

- [ ] **Step 3: Commit implementation**

Run:

```bash
git add skill_learning_proposals.go skill_learning_proposals_test.go migrations.go
git commit -m "feat: add skill learning proposal governance"
```

Expected: commit succeeds.

## Self-Review

- Spec coverage: Tasks cover SQLite audit trail, pending-only proposal creation, scoped pending listing, evidence JSON, approve/reject lifecycle, duplicate review failure, and migration inclusion.
- Placeholder scan: no unresolved TBD/TODO/FIXME markers are present.
- Type consistency: all test names and API names use the `SkillLearningProposal` prefix consistently.

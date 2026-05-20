# Skill Learning Governance Design

Date: 2026-05-20
Repo: `/home/xel/git/sages-openclaw/workspace-mineru/goncho`

## Goal

Add a Goncho-first primitive for governing agent self-learning before Gormes or any other runtime promotes generated skills. The improvement targets the real failure modes called out in the Hermes learning-loop discussion: self-evaluation bias, overfitting, weak retrieval evidence, missing audit trail, and accidental overwrite of human skills.

## Scope For Today's Slice

Implement a small SQLite-backed skill-learning proposal API. It records generated skill candidates as reviewable proposals with evidence and status. It does not create active `SKILL.md` files, run an LLM, select skills, or mutate Gormes.

## Data Model

Add a `skill_learning_proposals` table with:

- `proposal_id`: stable generated ID.
- `skill_name`: candidate skill name.
- `source_task`: task or session summary that produced the candidate.
- `draft_body`: proposed skill content or concise draft text.
- `evidence_json`: structured evidence such as tool-call count, validation commands, outcome, source session, and reviewer notes.
- `status`: `pending`, `approved`, or `rejected`.
- `created_by`: agent or system that created it.
- `reviewed_by`, `reviewed_at`, `review_reason`: explicit review trail.
- `created_at`: creation timestamp.

## API

Expose package-level functions and service methods similar to existing memory proposals:

- `SubmitSkillLearningProposal(ctx, db, params)` creates a pending proposal.
- `ListPendingSkillLearningProposals(ctx, db, workspaceID, limit)` lists unreviewed candidates.
- `GetSkillLearningProposal(ctx, db, proposalID)` fetches one proposal.
- `ApproveSkillLearningProposal(ctx, db, proposalID, reviewer, reason)` marks it approved.
- `RejectSkillLearningProposal(ctx, db, proposalID, reviewer, reason)` marks it rejected.

## Safety Rules

- Empty skill name, draft body, source task, or creator is rejected.
- Proposals start as `pending`; no API promotes directly to active skill files.
- Approved/rejected proposals cannot be reviewed again.
- Evidence is JSON and round-trips exactly enough for audit/debugging.
- The API is local and deterministic; no network, provider, or Gormes runtime dependency.

## Testing

Add focused Go tests that prove:

- valid proposal submission persists pending state and evidence;
- invalid proposals fail closed;
- pending list is scoped and ordered;
- approve/reject transitions require reviewer and reason;
- duplicate review attempts fail;
- evidence JSON round-trips.

Run:

```bash
go test ./... -count=1
```

## Non-Goals

- No auto-generation of `SKILL.md`.
- No semantic/vector retrieval changes.
- No Gormes wiring in this slice.
- No curator behavior changes.
- No manual skill overwrite behavior.

## Follow-Up

Gormes can later consume this primitive by writing background-review skill candidates into Goncho as pending proposals and excluding all pending/unapproved drafts from prompt injection.

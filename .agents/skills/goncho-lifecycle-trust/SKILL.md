---
name: goncho-lifecycle-trust
description: Use when implementing Goncho lifecycle, trust, temporal validity, supersession, review, audit, verification, or stale-memory behavior.
---

# Goncho Lifecycle and Trust

## Goal

Prevent stale, conflicting, low-confidence, or untrusted memory from silently steering agent behavior.

## Required TDD Shape

**REQUIRED SUB-SKILL:** Use `goncho-tdd-implementation` first. Each slice must prove one lifecycle behavior:

- temporal validity,
- supersession,
- conflict detection,
- review-required state,
- confidence or authority scoring,
- stale code/path verification,
- audit visibility,
- quarantine or redaction.

## Quick Reference

| Need | Prove with |
| --- | --- |
| Temporal validity | Expired or not-yet-valid memory is warned or suppressed |
| Supersession | Newer evidence changes current truth without deleting history |
| Conflict detection | Contradictory memories produce visible conflict state |
| Review workflow | Review-required items are queryable through API/tool output |
| Trust scoring | Confidence or authority changes ranking, warnings, or inclusion |
| Verification | File/path memory is checked before strong context injection |
| Quarantine | Secret or prompt-injection-like content is not promoted |

## Minimal Contract Examples

Write tests that prove user-visible behavior, for example:

- newer evidence supersedes older contradictory memory without deleting history,
- outdated memory is returned with a warning or suppressed by default,
- file/path memories require verification before strong context injection,
- review-required items are visible through a review API,
- secret-bearing or prompt-injection-like content is quarantined and not promoted.

## Data Rules

Lifecycle state is not enough by itself. A lifecycle feature must affect at least one of:

- storage mutation,
- search ranking,
- context-pack inclusion,
- review/audit output,
- API result warning fields.

## Done Criteria

- Historical evidence is preserved.
- Current truth is distinguishable from past truth.
- Context output explains stale/conflict/review warnings.
- `go test ./...` passes.

## Common Mistakes

| Mistake | Fix |
| --- | --- |
| Adding state fields that no behavior reads | Connect state to ranking, context, review, audit, or warnings |
| Overwriting old truth destructively | Preserve version/evidence history and mark current state |
| Silently dropping conflicts | Expose conflict state to search/context/review consumers |
| Treating confidence as decorative metadata | Make score affect inclusion, warning, ranking, or review state |

## Avoid

- destructive overwrites without version records,
- silently ignoring conflicts,
- adding review tables without APIs or tests,
- treating confidence as decorative metadata.

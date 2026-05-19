---
name: goncho-lifecycle-trust
description: Use when implementing Goncho lifecycle, trust, temporal validity, supersession, review, audit, verification, or stale-memory behavior.
---

# Goncho Lifecycle and Trust

## Goal

Prevent stale, conflicting, low-confidence, or untrusted memory from silently steering agent behavior.

## Required TDD Shape

Use `goncho-tdd-implementation` first. Each slice must prove one lifecycle behavior:

- temporal validity,
- supersession,
- conflict detection,
- review-required state,
- confidence or authority scoring,
- stale code/path verification,
- audit visibility,
- quarantine or redaction.

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

## Avoid

- destructive overwrites without version records,
- silently ignoring conflicts,
- adding review tables without APIs or tests,
- treating confidence as decorative metadata.

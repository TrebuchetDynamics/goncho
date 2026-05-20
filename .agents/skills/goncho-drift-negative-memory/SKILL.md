---
name: goncho-drift-negative-memory
description: Use when implementing Goncho negative memory, dead-end recall, drift detection, anchors, anti-repeat alerts, or feedback labels.
---

# Goncho Drift and Negative Memory

## Goal

Make memory prevent repeated mistakes, not only recall useful facts.

## Required TDD Shape

**REQUIRED SUB-SKILL:** Use `goncho-tdd-implementation` first. Each slice must prove one anti-repeat behavior:

- failed attempts are stored as negative memory,
- context packs surface relevant dead ends,
- drift anchors warn before repeated bad behavior,
- feedback labels reduce false positives,
- successful fixes supersede older failed paths without deleting them.

## Quick Reference

| Need | Prove with |
| --- | --- |
| Store a dead end | Failed action, evidence, scope, timestamp, and reason are persisted |
| Surface a warning | Relevant context pack includes the prior dead end before repetition |
| Detect drift | Anchor fires only for a matching repeated failure pattern |
| Reduce noise | Feedback label suppresses or demotes noisy alerts |
| Resolve old failures | Successful fix supersedes prior dead ends while preserving history |

## Minimal Contract Examples

Good tests:

- `TestContextSurfacesRelevantDeadEndBeforeRepeatingCommand`
- `TestNegativeAnchorWarnsOnRepeatedFailurePattern`
- `TestFeedbackLabelSuppressesFalsePositiveDriftAlert`
- `TestSuccessfulFixSupersedesPriorDeadEnds`

## Data Rules

Negative memory should record:

- attempted action,
- failure evidence,
- scope,
- timestamp,
- why it failed,
- what to do instead if known,
- supersession or resolution state.

## Context Rules

Dead ends should appear only when relevant to the current task. They must include enough evidence for the agent to avoid repeating the mistake.

## Done Criteria

- negative memory has tested storage and retrieval,
- at least one context path can surface it,
- alerts are scope-aware,
- feedback can mark useful or noisy alerts,
- `go test ./...` passes.

## Common Mistakes

| Mistake | Fix |
| --- | --- |
| Treating every transient error as durable memory | Require failure evidence, scope, and repeat-risk rationale |
| Emitting global warnings everywhere | Gate alerts by task, command, path, or entity relevance |
| Hiding resolved failures | Preserve history and mark supersession or resolution explicitly |
| Recording dead ends without retrieval tests | Add a context/search test that proves the warning is usable |

## Avoid

- global warnings that fire everywhere,
- hiding successful resolutions,
- treating every error as durable negative memory.

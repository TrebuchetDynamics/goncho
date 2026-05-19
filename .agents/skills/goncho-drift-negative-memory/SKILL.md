---
name: goncho-drift-negative-memory
description: Use when implementing Goncho negative memory, dead-end recall, drift detection, anchors, anti-repeat alerts, or feedback labels.
---

# Goncho Drift and Negative Memory

## Goal

Make memory prevent repeated mistakes, not only recall useful facts.

## Required TDD Shape

Use `goncho-tdd-implementation` first. Each slice must prove one anti-repeat behavior:

- failed attempts are stored as negative memory,
- context packs surface relevant dead ends,
- drift anchors warn before repeated bad behavior,
- feedback labels reduce false positives,
- successful fixes supersede older failed paths without deleting them.

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

## Avoid

- global warnings that fire everywhere,
- hiding successful resolutions,
- treating every error as durable negative memory.

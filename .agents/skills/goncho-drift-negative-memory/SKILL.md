---
name: goncho-drift-negative-memory
description: Implement Goncho negative memory and anti-repeat behavior. Use when working on dead-end recall, drift anchors, repeated-failure warnings, feedback labels, or negative-memory supersession.
---

# Goncho Drift and Negative Memory

## Quick start

Load `goncho-tdd-implementation`, then prove one anti-repeat behavior with a failing test before changing code.

## Workflow

1. **Choose one repeat-risk**
   - Failed command/action, stale path, bad plan pattern, noisy alert, or resolved dead end.
   - Record scope: workspace, profile, session, path, command, entity, or task pattern.
2. **Write the contract test**
   - Good names: `TestContextSurfacesRelevantDeadEndBeforeRepeatingCommand`, `TestNegativeAnchorWarnsOnRepeatedFailurePattern`, `TestFeedbackLabelSuppressesFalsePositiveDriftAlert`, `TestSuccessfulFixSupersedesPriorDeadEnds`.
3. **Implement minimally**
   - Persist failure evidence, reason, timestamp, scope, and optional alternative.
   - Surface warnings only when relevant to the current task.
   - Preserve resolved failures with supersession/resolution state.
4. **Verify**
   - Run the narrow test, then `go test ./...`.
   - Update docs/status if the public warning or feedback behavior changed.

## Data rules

Negative memory should include attempted action, failure evidence, scope, timestamp, why it failed, what to do instead if known, and supersession or resolution state.

## Skill contract

### Entry protocol
- Trivial: answer design questions or inspect existing dead-end behavior directly.
- Medium ambiguity: propose the smallest anti-repeat slice and ask only what failure pattern matters most.
- High ambiguity/risk: stop if the requested behavior would globally suppress memories, hide evidence, or mutate unrelated review/lifecycle state.

### Topology check
- State/ownership: where dead ends, anchors, feedback labels, and supersession live.
- Feedback/validation: storage proof plus search/context/warning proof.
- Blast radius: warning noise, false suppression, review queues, imports, and recall ranking.
- Timing/ordering: repeated attempts, feedback updates, and successful-fix supersession.

### Verification gate
Done requires tested storage/retrieval, at least one user-visible context/search/warning path, scope-aware alerts, and `go test ./...` pass or blocker output.

### Red lines
- Do not turn every transient error into durable negative memory.
- Do not emit global warnings for local failures.
- Do not delete or hide resolved failures; preserve history with state.
- Do not add alerts without feedback/noise handling when false positives are plausible.

### Output contract
End with: failure pattern covered, test names, warning/suppression behavior, validation commands, and remaining false-positive risks.

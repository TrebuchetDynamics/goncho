---
name: goncho-grill-me
description: Stress-test Goncho-specific implementation plans before coding. Use when asked to grill a Goncho roadmap slice, validate readiness for TDD, challenge a memory-system design, or find gaps against Goncho docs/metaanalysis.
---

# Goncho Grill Me

## Quick start

Inspect relevant Goncho docs/code first, then ask one hard question at a time with a recommended answer.

Use generic `grill-me` for non-Goncho plans. Use this skill only when the plan must fit Goncho's memory architecture, roadmap, tests, or metaanalysis.

## Workflow

1. **Ground the plan**
   - Identify the TODO/README/docs/metaanalysis capability.
   - Inspect current files/tests instead of asking questions the repo can answer.
   - Name the current implementation status and public contract/API surface.
2. **Challenge the design**
   - Capability fit: what gap closes?
   - Trust model: evidence, scope, time, confidence, authority, lifecycle.
   - User-visible behavior: search, context, review, audit, tool, HTTP, or CLI output.
   - Failure mode: stale, conflicting, leaking, noisy, or repeated-bad-memory case.
   - TDD proof: first failing test and why it fails now.
   - Minimal slice: smallest vertical behavior.
   - Non-goals: tempting phase work excluded.
3. **Stop before implementation**
   - If implementation is approved, switch to `goncho-tdd-implementation`.

## Question format

```text
Question N: <specific challenge>
Why it matters: <risk or design dependency>
Recommended answer: <concrete default>
```

Ask only one decision-driving question at a time.

## Skill contract

### Entry protocol
- Trivial: if the plan is already concrete, summarize readiness and the first failing test.
- Medium ambiguity: propose the likely slice and ask the single hard owner decision.
- High ambiguity/risk: stop if ownership, trust boundary, or destructive/public behavior is unclear.

### Topology check
- State/ownership: memory data, evidence, lifecycle, host authority, and adapter boundaries.
- Feedback/validation: exact RED test, user-visible output, and full verification command.
- Blast radius: public API/tool behavior, migrations, benchmarks, docs, server/team mode.
- Timing/ordering: imports, recalls, review, supersession, concurrency, release sequencing.

### Verification gate
A grill session is done when it names the chosen slice, first failing test, likely files, risks/non-goals, and asks for approval before TDD.

### Red lines
- Do not implement during grilling.
- Do not ask broad questionnaires; inspect first and ask one hard question.
- Do not accept schema-only completion for user-visible features.
- Do not expand one slice into a whole roadmap phase.

### Output contract
End with: chosen slice, first failing test, likely files, risks/non-goals, and explicit ask to start TDD.

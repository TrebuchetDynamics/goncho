---
name: goncho-grill-me
description: Stress-tests Goncho plans against the metaanalysis, feature matrix, trust model, and TDD requirements. Use when user says grill me, stress-test this, challenge this plan, or asks whether a Goncho design is ready to implement.
---

# Goncho Grill Me

## Purpose

Interrogate a Goncho design until the implementation slice is clear, testable, and aligned with `docs/opensource-memory-systems/METAANALYSIS-MEMORY-SYSTEMS.md`.

Ask one question at a time. For each question, include your recommended answer.

If a question can be answered by reading the codebase, inspect the codebase instead of asking.

## Required Context

Before grilling, identify:

- target metaanalysis capability,
- current implementation status,
- relevant files and tests,
- public contract or API surface,
- smallest TDD slice.

Use `goncho-tdd-implementation` before turning answers into code.

## Grill Sequence

1. **Capability fit** — Which metaanalysis gap does this close?
2. **Trust model** — What evidence, scope, time, confidence, or authority does it need?
3. **User-visible behavior** — What changes in search, context, review, audit, or tools?
4. **Failure mode** — What stale, conflicting, leaking, or noisy memory case must be prevented?
5. **TDD proof** — What is the first failing test, and why should it fail now?
6. **Minimal slice** — What can be shipped without building the whole phase?
7. **Non-goals** — What tempting architecture work is explicitly out of scope?

## Question Format

```text
Question N: <specific challenge>
Why it matters: <risk or design dependency>
Recommended answer: <concrete answer or default>
```

Stop when each answer produces a small implementation contract that can be tested.

## Approval Gate

Do not implement during grilling. End with:

- chosen slice,
- first failing test name,
- files likely touched,
- risks,
- explicit ask for approval to start TDD.

## Avoid

- asking multiple questions at once,
- accepting schema-only answers as feature completion,
- skipping code inspection when the answer is on disk,
- expanding one slice into a full metaanalysis phase.

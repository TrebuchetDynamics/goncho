---
name: goncho-grill-me
description: Use when stress-testing Goncho plans, challenging Goncho designs, checking readiness to implement, or responding to user requests to grill me.
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

**REQUIRED SUB-SKILL:** Use `goncho-tdd-implementation` before turning answers into code.

## Quick Reference

| Need | Ask or inspect |
| --- | --- |
| Capability fit | Which metaanalysis gap or roadmap item this closes |
| Trust model | Evidence, scope, time, confidence, authority, lifecycle state |
| User-visible behavior | Search, context, review, audit, tool, or API change |
| Failure mode | Stale, conflicting, leaking, noisy, or repeated-bad-memory case |
| TDD proof | First failing test and why it fails now |
| Minimal slice | Smallest shippable vertical behavior |
| Non-goals | Architecture or phase work explicitly excluded |

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

## Done Criteria

- relevant code/docs were inspected before asking answerable questions,
- each question is single-focus and includes a recommended answer,
- final slice has one failing test name and likely touched files,
- implementation is not started during grilling,
- approval is requested before TDD begins.

## Approval Gate

Do not implement during grilling. End with:

- chosen slice,
- first failing test name,
- files likely touched,
- risks,
- explicit ask for approval to start TDD.

## Common Mistakes

| Mistake | Fix |
| --- | --- |
| Asking several broad questions at once | Ask one decision-driving question at a time |
| Asking what the repo can answer | Inspect files/tests first and cite the evidence |
| Treating schema as feature completion | Demand user-visible behavior and a failing contract test |
| Letting one slice become a whole phase | Name non-goals and choose the smallest vertical proof |

## Avoid

- asking multiple questions at once,
- accepting schema-only answers as feature completion,
- skipping code inspection when the answer is on disk,
- expanding one slice into a full metaanalysis phase.

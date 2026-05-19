---
title: Evidence, Claims, and Beliefs
description: The vocabulary behind trust-preserving memory.
---

Chunks are not memory. Summaries are not automatically truth. A memory system needs a way to separate what happened from what it currently believes.

## Evidence

Evidence is raw material: transcripts, tool outputs, user statements, session events, observations, and imported files.

Evidence should survive changes in interpretation.

## Claims

A claim is a distilled statement derived from evidence.

Examples:

- "The user prefers SQLite over Postgres."
- "This deployment strategy failed twice."
- "This peer card is from the assistant's perspective."

## Beliefs

A belief is a claim accepted for a scope and time. Beliefs can be revised when evidence conflicts, ages out, or becomes less relevant.

:::note[Conceptual example]
Goncho's current peer cards and conclusions are early belief products. First-class claim lifecycle states are architecture direction.
:::

---
title: Trust-Preserving Context
description: Goncho's core memory model.
---

Memory is not a vector database problem.

For a long-running agent, memory is the state it carries forward: what it believes, why it believes it, where that belief applies, and when it may be wrong.

Goncho's architecture direction is:

```text
raw evidence
  -> derived claims
  -> scoped beliefs
  -> retrieved orientation
  -> agent action
  -> consolidation / revision / forgetting
```

## Why Context Instead Of Dumps

Dumping every remembered fact into a prompt increases token cost and reasoning noise. The agent needs orientation: current constraints, trusted facts, warnings, active goals, recent drift, and unresolved conflicts.

## Trust Constraint

A memory system should explain why a fact surfaced. It should also preserve enough evidence to revise or remove that fact when the world changes.

:::note[Current status]
Current Goncho APIs expose peer cards, search, context assembly, session summaries, and memory tools. First-class claim state, confidence, and review products are direction.
:::

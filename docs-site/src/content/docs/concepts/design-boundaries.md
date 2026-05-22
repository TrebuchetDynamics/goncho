---
title: Design Boundaries
description: What Goncho is and is not.
---

Goncho is not a vector database, hosted memory SaaS, app database, or agent framework.

It is a local-first context layer for Go agents.

## RAG And Memory

RAG retrieves external context. Memory maintains state across time.

Vector search can help find evidence, but a retrieved chunk is not automatically a belief. Long-running agents need scope, time, provenance, and revision.

## Storage Boundary

SQLite is Goncho's current local persistence foundation. Use [Goncho APIs](/reference/core-api/) and [memory tools](/reference/memory-tools/) for reads and writes unless a storage contract is explicitly documented.

:::caution[Failure mode]
Globally true forever memories corrupt long-running agents. Facts need scope and revision.
:::

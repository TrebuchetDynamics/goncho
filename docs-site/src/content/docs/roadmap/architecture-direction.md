---
title: Architecture Direction
description: Design constraints shaping Goncho beyond the current pre-release API.
---

Goncho's direction is trust-preserving context architecture.

The roadmap is framed as design constraints first and feature promises second.

| Constraint | Current Representation | Direction |
| --- | --- | --- |
| Evidence must remain inspectable | session turns, markdown memory, search hits | stronger evidence lineage |
| Derived beliefs must be revisable | peer cards, conclusions, updates, forget tools | review and repair workflows |
| Context should orient the agent | `Context` result shape | compact orientation packs |
| Scope prevents false universals | workspaces, peers, observers, sessions | richer scoped belief model |
| Failures are part of intelligence | storable as memory or conclusions | first-class negative memory |
| Optional adapters stay optional | no mandatory cloud dependency | pluggable consolidation and recall |

Related concept pages expand the same constraints: [Trust-Preserving Context](/concepts/trust-preserving-context/), [Evidence, Claims, and Beliefs](/concepts/evidence-claims-beliefs/), [Orientation Packs](/concepts/orientation-packs/), [Negative Memory](/concepts/negative-memory/), and [Current Capabilities](/start/current-capabilities/).

## Not Yet Stable API

These terms guide architecture but are not stable exported APIs yet:

- first-class claim lifecycle;
- quantitative confidence;
- validity windows;
- review queues;
- memory repair products;
- handoff products.

The docs should name these constraints without pretending they are already complete.

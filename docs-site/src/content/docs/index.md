---
title: Goncho
description: Trust-preserving context for Go agents.
template: splash
hero:
  tagline: Trust-preserving context for Go agents.
  actions:
    - text: Start using Goncho
      link: /start/quick-start/
      icon: right-arrow
    - text: Understand the architecture
      link: /concepts/trust-preserving-context/
      icon: open-book
---

Goncho is a local-first context architecture for Go agents that preserves evidence, derives scoped beliefs, and returns compact orientation instead of dumping memory into prompts.

```text
raw evidence -> claims -> scoped beliefs -> orientation -> action -> revision
```

## Memory Is Not Just Retrieval

Vector search can help find relevant text. It does not decide what an agent should believe, where that belief applies, or when it may be stale.

Goncho treats memory as the state an agent carries forward: what it knows about peers, what it has concluded from prior sessions, what failed before, and what should be surfaced now.

:::note[Current status]
Goncho is pre-1.0. Public `@latest` currently resolves to v0.1.0, and the v0.1.x Go library supports local persistence, peer cards, search, context assembly, session summaries, local markdown memory, public tools, trust checks, and compatibility surfaces. Deeper graph/cognitive-map layers remain architecture direction.

Public API reference: [pkg.go.dev/github.com/TrebuchetDynamics/goncho](https://pkg.go.dev/github.com/TrebuchetDynamics/goncho).

The root module is a library package, not a CLI binary. Use `go get github.com/TrebuchetDynamics/goncho@latest` for the library; use checkout-local benchmark commands until a v0.1.x tag includes `goncho-bench`.

Local ecosystem smoke: `make ecosystem-smoke` verifies public release metadata, package docs, external importability, and checkout-local benchmark CLI installation. For the narrower public release metadata proof, run `make public-release-smoke`; for the narrower external import proof, run `make public-module-smoke`.
:::

## Four Paths

- Start with [Quick Start](/start/quick-start/) if you want the current Go integration shape.
- Start with [Trust-Preserving Context](/concepts/trust-preserving-context/) if you want the architecture model first.
- Start with [Operator Runbook](/operators/runbook/) if you need deployment, backup, health, and troubleshooting guidance.
- Start with [Gormes Agent Integration](/integrations/gormes-agent/) if you want to plug Goncho into a Gormes-style agent host.

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
Goncho is pre-1.0. Public `@latest` currently resolves to v0.3.0, published May 25, 2026, and the v0.3.x Go library supports local persistence, peer cards, search, context assembly, session summaries, local markdown memory, public tools, trust checks, and compatibility surfaces. Deeper graph/cognitive-map layers remain architecture direction.

Public API reference: [pkg.go.dev/github.com/TrebuchetDynamics/goncho/service](https://pkg.go.dev/github.com/TrebuchetDynamics/goncho/service).

The service package is a library package, not a CLI binary and not a root `go install` target. Use `go get github.com/TrebuchetDynamics/goncho/service@latest` for the library; use `go install github.com/TrebuchetDynamics/goncho/cmd/goncho-bench@latest` or checkout-local benchmark commands for `goncho-bench`.

Local ecosystem smoke: `make ecosystem-smoke` verifies public release metadata, local Go module metadata, package docs, public docs site build, external importability, and checkout-local benchmark CLI installation. For the narrower public release metadata proof, run `make public-release-smoke`; it checks the documented public `@latest` version and published date. For the narrower local go.mod metadata proof, run `make local-module-smoke`; for the narrower package documentation proof, run `make package-doc-smoke`; for the narrower public docs site proof, run `make docs-site-smoke`; for the narrower external import proof, run `make public-module-smoke`.

Benchmark methodology, the external adapter contract, and current agentmemory PR #583 stable-ID status live in [Retrieval Benchmarks](/reference/retrieval-benchmarks/). For the CI-safe external backend comparison proof, run `make bench-locomo-backends-smoke` from a checkout.
:::

## Four Paths

- Start with [Quick Start](/start/quick-start/) if you want the current Go integration shape.
- Start with [Trust-Preserving Context](/concepts/trust-preserving-context/) if you want the architecture model first.
- Start with [Operator Runbook](/operators/runbook/) if you need deployment, backup, health, and troubleshooting guidance.
- Start with [Gormes Agent Integration](/integrations/gormes-agent/) if you want to plug Goncho into a Gormes-style agent host.

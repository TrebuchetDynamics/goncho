# Hermes Connector

Status: deferred

Connector status contract: deferred, local-first preview only; no install automation before `goncho-server` smoke coverage.

Shared deferred connector status contract: [Deferred Connector Readiness](../contracts/deferred-readiness.md).

Hermes connector docs are intentionally deferred. Use the Gormes adapter path when Hermes is operating through Gormes, and keep Goncho local-first via `goncho-server` on loopback.

Before this becomes supported, add a preview plan, golden config tests, hook-event mapping tests, and a host smoke. Do not publish install instructions for unsupported Hermes mutation flows.

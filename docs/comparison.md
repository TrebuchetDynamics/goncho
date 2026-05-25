# Goncho vs mem0 vs agentmemory

This comparison is for product fit, not hype. No star-count ranking, popularity scoring, or benchmark overclaims should decide whether Goncho is right for a host.

## Goncho

Goncho is local-first, Go-native memory infrastructure. It prioritizes evidence, provenance, scoped recall, review state, lifecycle state, local SQLite operation, deterministic smoke tests, and preview-first operator workflows.

Choose Goncho when:

- the host must keep memory local by default;
- retrieved memory must explain where evidence came from;
- review, stale warnings, redaction, import/export, and audit trails matter;
- Go services or local agent hosts need embedded memory without a hosted sidecar.

## mem0

mem0 has a compact product API shape: add/search/update/delete/history. Goncho adapts that lesson through a local facade while preserving evidence and review state. Goncho does not copy hosted/cloud assumptions or hide provenance behind a simple success response.

## agentmemory

agentmemory has broad connector, server, retention, provider, and operational UX ideas. Goncho adapts the useful patterns cautiously: local-first connector plans, MCP resources/prompts, provider fallback diagnostics, retention previews, portable exports, and server-mode threat models.

## Benchmark claims

Benchmark claims require reproducible commands, fixed datasets, failure artifacts, and clear scope. LOCOMO/LongMemEval/BEAM results in this repo are evidence for specific retrieval configurations, not universal proof that one memory system is better for every workload.

Use the benchmark docs and failure audits to decide what to improve next; do not make broad marketing claims from a single score.

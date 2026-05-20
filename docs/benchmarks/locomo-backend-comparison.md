# LOCOMO External Backend Comparison

This is a benchmark adapter suite, not a marketing dunk. It compares retrieval backends only when they can return stable inserted memory IDs.

- JSON evidence: `./docs/benchmarks/results/locomo-backend-comparison.json`
- Failure JSONL: `./docs/benchmarks/failures/locomo-backend-comparison.jsonl`
- Memories: `./data/locomo/memories.jsonl`
- Questions: `./data/locomo/questions.jsonl`
- Questions: `1982`
- Memories: `5882`
- no_llm_judge: `true`

## Rules

- retrieval only
- no LLM judge
- no answer generation
- same converted memories/questions
- same gold memory IDs
- same top-K scoring
- if stable memory IDs are unavailable, mark backend not comparable

## Results

| Backend | Comparable | recall_any@5 | recall_any@10 | strict@5 | strict@10 | MRR | Search latency ms | Notes |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| `goncho` | true | 60.14% | 67.91% | 51.16% | 57.67% | 46.89% | 21194 | Local deterministic adapter in cmd/goncho-bench. |
| `bm25` | true | 60.14% | 68.11% | 51.21% | 57.97% | 46.91% | 12397 | Local deterministic adapter in cmd/goncho-bench. |
| `sqlite-fts5` | true | 49.24% | 56.31% | 42.03% | 48.28% | 38.17% | 6908 | Local deterministic adapter in cmd/goncho-bench. |
| `agentmemory` | true | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0 | External Python probe: scripts/bench_agentmemory_locomo.py --capability. Comparable when AGENTMEMORY_SOURCE_DIR points at PR #583 / commit 9b18a80c9d2839b025279978d3f4b5e1f9bc6e74 with npm dependencies installed. Adapter path uses standalone InMemoryKV fallback: memory_save external_id plus metadata.memory_id, then memory_smart_search. This validates stable IDs but is not the full running agentmemory server. If AGENTMEMORY_SOURCE_DIR is absent, agentmemory is marked not comparable. |
| `mem0` | false | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0 | not comparable: Python package mem0/mem0ai is not installed in this environment |

## Setup notes

- Goncho, BM25, and SQLite FTS5 are local Go adapters with no hosted dependency.
- agentmemory probe: `python3 scripts/bench_agentmemory_locomo.py --capability`. Comparable when `AGENTMEMORY_SOURCE_DIR` points at PR #583 / commit `9b18a80c9d2839b025279978d3f4b5e1f9bc6e74` with npm dependencies installed. This adapter uses the standalone InMemoryKV fallback, not the full running agentmemory server.
- mem0 probe: `python3 scripts/bench_mem0_locomo.py --capability`. Exact package version used here: none; backend is marked not comparable before scoring. Candidate install: `pip install mem0ai` plus upstream local vector-store dependencies. Comparable only after configured local retrieval can return caller-supplied `memory_id` without answer-generation scoring.

## Interpretation

Backends marked not comparable are excluded from score claims until they implement the `MemoryBackend` contract and return the same stable `memory_id` values that were inserted. This keeps the arena fair and prevents answer-generation or LLM-judge effects from leaking into retrieval metrics.

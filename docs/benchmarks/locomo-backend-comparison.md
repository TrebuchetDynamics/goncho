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
| `goncho` | true | 60.14% | 67.91% | 51.16% | 57.67% | 46.89% | 17777 | Local deterministic adapter in cmd/goncho-bench. |
| `bm25` | true | 60.14% | 68.06% | 51.21% | 57.92% | 46.91% | 12084 | Local deterministic adapter in cmd/goncho-bench. |
| `sqlite-fts5` | true | 49.24% | 56.31% | 42.03% | 48.28% | 38.17% | 7022 | Local deterministic adapter in cmd/goncho-bench. |
| `agentmemory` | false | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0 | not comparable: no stable-memory-id LOCOMO adapter is wired for agentmemory; scoring requires search results to return the inserted memory_id exactly |
| `mem0` | false | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0 | not comparable: no stable-memory-id LOCOMO adapter is wired for mem0; scoring requires search results to return the inserted memory_id exactly |

## Setup notes

- Goncho, BM25, and SQLite FTS5 are local Go adapters with no hosted dependency.
- agentmemory probe: `python3 scripts/bench_agentmemory_locomo.py --capability`. Exact package version used here: none; backend is marked not comparable before scoring. Comparable only after a local adapter can reset state, insert caller-supplied `memory_id`, and return that same ID from retrieval.
- mem0 probe: `python3 scripts/bench_mem0_locomo.py --capability`. Exact package version used here: none; backend is marked not comparable before scoring. Candidate install: `pip install mem0ai` plus upstream local vector-store dependencies. Comparable only after configured local retrieval can return caller-supplied `memory_id` without answer-generation scoring.

## Interpretation

Backends marked not comparable are excluded from score claims until they implement the `MemoryBackend` contract and return the same stable `memory_id` values that were inserted. This keeps the arena fair and prevents answer-generation or LLM-judge effects from leaking into retrieval metrics.

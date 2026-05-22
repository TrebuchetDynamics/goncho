# LOCOMO External Backend Comparison

This is a benchmark adapter suite, not a marketing dunk. It compares retrieval backends only when they can return stable inserted memory IDs.

- JSON evidence: `./docs/benchmarks/results/locomo-backend-comparison-smoke.json`
- Failure JSONL: `./docs/benchmarks/failures/locomo-backend-comparison-smoke.jsonl`
- Memories: `./cmd/goncho-bench/testdata/locomo-smoke/memories.jsonl`
- Questions: `./cmd/goncho-bench/testdata/locomo-smoke/questions.jsonl`
- Questions: `8`
- Memories: `17`
- Memory token estimate: `143`
- Database size bytes: `7314`
- Top-K: `10`
- no_llm_judge: `true`
- Reproduce: `go run ./cmd/goncho-bench --locomo-memories ./cmd/goncho-bench/testdata/locomo-smoke/memories.jsonl --locomo-questions ./cmd/goncho-bench/testdata/locomo-smoke/questions.jsonl --locomo-backend-comparison-json-out ./docs/benchmarks/results/locomo-backend-comparison-smoke.json --locomo-backend-comparison-md-out ./docs/benchmarks/locomo-backend-comparison-smoke.md --locomo-backend-comparison-failures-out ./docs/benchmarks/failures/locomo-backend-comparison-smoke.jsonl --limit 10`

## Leakage checks

- Answer text present in memory content: `12`
- Gold IDs present in memory content: `0`
- Question text present in memory content: `0`

## Rules

- retrieval only
- no LLM judge
- no answer generation
- same converted memories/questions
- same gold memory IDs
- same top-K scoring
- if stable memory IDs are unavailable, mark backend not comparable

## Results

| Backend | Comparable | recall_any@5 | recall_any@10 | strict@5 | strict@10 | NDCG@5 | NDCG@10 | MRR | Insert latency ms | Search latency ms | Latency min ms | Latency p50 ms | Latency p95 ms | Latency max ms | RSS bytes | Notes |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| `goncho` | true | 100.00% | 100.00% | 100.00% | 100.00% | 87.60% | 87.60% | 85.42% | 3 | 1 | 0 | 0 | 0 | 0 | 12278024 | Local deterministic adapter in cmd/goncho-bench. |
| `goncho-no-rank` | true | 75.00% | 100.00% | 75.00% | 100.00% | 52.64% | 60.20% | 46.98% | 0 | 0 | 0 | 0 | 0 | 0 | 12278024 | Local deterministic no-ranking baseline in cmd/goncho-bench; uses the recency order before current Goncho ranking. |
| `bm25` | true | 100.00% | 100.00% | 100.00% | 100.00% | 87.27% | 87.27% | 85.42% | 0 | 0 | 0 | 0 | 0 | 0 | 12278024 | Local deterministic adapter in cmd/goncho-bench. |
| `sqlite-fts5` | true | 87.50% | 100.00% | 87.50% | 100.00% | 80.25% | 84.70% | 81.25% | 3 | 2 | 0 | 0 | 0 | 0 | 12278024 | Local deterministic adapter in cmd/goncho-bench. |
| `agentmemory` | true | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0 | 0 | 0 | 0 | 0 | 0 | 0 | External Python probe: scripts/bench_agentmemory_locomo.py --capability. Comparable when AGENTMEMORY_SOURCE_DIR points at PR #583 / commit 9b18a80c9d2839b025279978d3f4b5e1f9bc6e74 with npm dependencies installed. Adapter path uses standalone InMemoryKV fallback: memory_save external_id plus metadata.memory_id, then memory_smart_search. This validates stable IDs but is not the full running agentmemory server. If AGENTMEMORY_SOURCE_DIR is absent, agentmemory is marked not comparable. |
| `mem0` | false | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0 | 0 | 0 | 0 | 0 | 0 | 0 | not comparable: Python package mem0/mem0ai is not installed in this environment |

## Failure categories

| Backend | Category | Questions |
| --- | --- | ---: |
| `goncho` | `gold_not_rank_1` | 2 |
| `goncho` | `gold_rank_1` | 6 |
| `goncho-no-rank` | `gold_not_rank_1` | 6 |
| `goncho-no-rank` | `gold_rank_1` | 2 |
| `bm25` | `gold_not_rank_1` | 2 |
| `bm25` | `gold_rank_1` | 6 |
| `sqlite-fts5` | `gold_not_rank_1` | 2 |
| `sqlite-fts5` | `gold_rank_1` | 6 |
| `agentmemory` | `miss_top_10` | 8 |

## Category metrics

### goncho

| Category | Questions | recall_any@5 | recall_any@10 | strict@5 | strict@10 | NDCG@5 | NDCG@10 | MRR |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `contradiction_supersession` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 63.09% | 63.09% | 50.00% |
| `gold_ambiguity` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `historical_retrieval` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 50.00% | 50.00% | 33.33% |
| `latest_state_retrieval` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `lexical_miss` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `multi_session_continuity` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 87.72% | 87.72% | 100.00% |
| `speaker_attribution` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `true_retrieval_failure` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |

### goncho-no-rank

| Category | Questions | recall_any@5 | recall_any@10 | strict@5 | strict@10 | NDCG@5 | NDCG@10 | MRR |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `contradiction_supersession` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 38.69% | 38.69% | 20.00% |
| `gold_ambiguity` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `historical_retrieval` | 1 | 0.00% | 100.00% | 0.00% | 100.00% | 0.00% | 28.91% | 10.00% |
| `latest_state_retrieval` | 1 | 0.00% | 100.00% | 0.00% | 100.00% | 0.00% | 31.55% | 12.50% |
| `lexical_miss` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 69.34% | 69.34% | 50.00% |
| `multi_session_continuity` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `speaker_attribution` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 50.00% | 50.00% | 33.33% |
| `true_retrieval_failure` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 63.09% | 63.09% | 50.00% |

### bm25

| Category | Questions | recall_any@5 | recall_any@10 | strict@5 | strict@10 | NDCG@5 | NDCG@10 | MRR |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `contradiction_supersession` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 63.09% | 63.09% | 50.00% |
| `gold_ambiguity` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `historical_retrieval` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 50.00% | 50.00% | 33.33% |
| `latest_state_retrieval` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `lexical_miss` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `multi_session_continuity` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 85.03% | 85.03% | 100.00% |
| `speaker_attribution` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `true_retrieval_failure` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |

### sqlite-fts5

| Category | Questions | recall_any@5 | recall_any@10 | strict@5 | strict@10 | NDCG@5 | NDCG@10 | MRR |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `contradiction_supersession` | 1 | 0.00% | 100.00% | 0.00% | 100.00% | 0.00% | 35.62% | 16.67% |
| `gold_ambiguity` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `historical_retrieval` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 50.00% | 50.00% | 33.33% |
| `latest_state_retrieval` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `lexical_miss` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `multi_session_continuity` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 91.97% | 91.97% | 100.00% |
| `speaker_attribution` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `true_retrieval_failure` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |

### agentmemory

| Category | Questions | recall_any@5 | recall_any@10 | strict@5 | strict@10 | NDCG@5 | NDCG@10 | MRR |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `contradiction_supersession` | 1 | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% |
| `gold_ambiguity` | 1 | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% |
| `historical_retrieval` | 1 | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% |
| `latest_state_retrieval` | 1 | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% |
| `lexical_miss` | 1 | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% |
| `multi_session_continuity` | 1 | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% |
| `speaker_attribution` | 1 | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% |
| `true_retrieval_failure` | 1 | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% |

## Setup notes

- Goncho, Goncho no-rank, BM25, and SQLite FTS5 are local Go adapters with no hosted dependency.
- agentmemory probe: `python3 scripts/bench_agentmemory_locomo.py --capability`. Comparable when `AGENTMEMORY_SOURCE_DIR` points at PR #583 / commit `9b18a80c9d2839b025279978d3f4b5e1f9bc6e74` with npm dependencies installed. This adapter uses the standalone InMemoryKV fallback, not the full running agentmemory server.
- mem0 probe: `python3 scripts/bench_mem0_locomo.py --capability`. Exact package version used here: none; backend is marked not comparable before scoring. Candidate install: `pip install mem0ai` plus upstream local vector-store dependencies. Comparable only after configured local retrieval can return caller-supplied `memory_id` without answer-generation scoring.

## Interpretation

Backends marked not comparable are excluded from score claims until they implement the `MemoryBackend` contract and return the same stable `memory_id` values that were inserted. This keeps the arena fair and prevents answer-generation or LLM-judge effects from leaking into retrieval metrics.

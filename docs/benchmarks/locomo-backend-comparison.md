# LOCOMO External Backend Comparison

This is a benchmark adapter suite, not a marketing dunk. It compares retrieval backends only when they can return stable inserted memory IDs.

- JSON evidence: `./docs/benchmarks/results/locomo-backend-comparison.json`
- Failure JSONL: `./docs/benchmarks/failures/locomo-backend-comparison.jsonl`
- Memories: `./data/locomo/memories.jsonl`
- Questions: `./data/locomo/questions.jsonl`
- Questions: `1982`
- Memories: `5882`
- Memory token estimate: `139594`
- Database size bytes: `3247476`
- Source: `https://github.com/snap-research/locomo` at `3eb6f2c585f5e1699204e3c3bdf7adc5c28cb376`
- Source SHA256: `79fa87e90f04081343b8c8debecb80a9a6842b76a7aa537dc9fdf651ea698ff4`
- Converted memories SHA256: `bd24ddbebb21e3dfeb9108c4f869048afc8d0425003424b37630bde1b35b48ff`
- Converted questions SHA256: `904c90f99963b9744117d4bfabd5f7570044c94d014c8b05a42ff444af27e5cd`
- License note: `Creative Commons Attribution-NonCommercial 4.0 International (CC BY-NC 4.0)`
- Top-K: `10`
- no_llm_judge: `true`
- Reproduce: `go run ./cmd/goncho-bench --locomo-memories ./data/locomo/memories.jsonl --locomo-questions ./data/locomo/questions.jsonl --locomo-backend-comparison-json-out ./docs/benchmarks/results/locomo-backend-comparison.json --locomo-backend-comparison-md-out ./docs/benchmarks/locomo-backend-comparison.md --locomo-backend-comparison-failures-out ./docs/benchmarks/failures/locomo-backend-comparison.jsonl --limit 10`

## Leakage checks

- Answer text present in memory content: `3026`
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
| `goncho` | true | 60.14% | 67.96% | 51.16% | 57.72% | 46.63% | 49.11% | 46.90% | 1611 | 23405 | 6 | 11 | 17 | 21 | 47155480 | Local deterministic adapter in cmd/goncho-bench. |
| `goncho-no-rank` | true | 0.25% | 1.11% | 0.15% | 0.96% | 0.10% | 0.36% | 0.20% | 6 | 414 | 0 | 0 | 0 | 1 | 51349784 | Local deterministic no-ranking baseline in cmd/goncho-bench; uses the recency order before current Goncho ranking. |
| `bm25` | true | 60.14% | 68.06% | 51.21% | 57.92% | 46.62% | 49.15% | 46.90% | 4 | 11731 | 3 | 6 | 7 | 10 | 55839000 | Local deterministic adapter in cmd/goncho-bench. |
| `sqlite-fts5` | true | 49.24% | 56.31% | 42.03% | 48.28% | 37.91% | 40.20% | 38.17% | 1166 | 7542 | 0 | 3 | 9 | 33 | 55839000 | Local deterministic adapter in cmd/goncho-bench. |
| `agentmemory` | true | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0 | 0 | 0 | 0 | 0 | 0 | 0 | External Python probe: scripts/bench_agentmemory_locomo.py --capability. Comparable when AGENTMEMORY_SOURCE_DIR points at https://github.com/rohitg00/agentmemory PR #583 / commit 9b18a80c9d2839b025279978d3f4b5e1f9bc6e74 with npm dependencies installed. Adapter path uses standalone InMemoryKV fallback: memory_save external_id plus metadata.memory_id, then memory_smart_search. This validates stable IDs but is not the full running agentmemory server. If AGENTMEMORY_SOURCE_DIR is absent, agentmemory is marked not comparable. |
| `mem0` | false | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0 | 0 | 0 | 0 | 0 | 0 | 0 | not comparable: Python package mem0/mem0ai is not installed in this environment |

## Failure categories

| Backend | Category | Questions |
| --- | --- | ---: |
| `goncho` | `gold_not_rank_1` | 613 |
| `goncho` | `gold_rank_1` | 734 |
| `goncho` | `miss_top_10` | 635 |
| `goncho-no-rank` | `gold_not_rank_1` | 21 |
| `goncho-no-rank` | `gold_rank_1` | 1 |
| `goncho-no-rank` | `miss_top_10` | 1960 |
| `bm25` | `gold_not_rank_1` | 616 |
| `bm25` | `gold_rank_1` | 733 |
| `bm25` | `miss_top_10` | 633 |
| `sqlite-fts5` | `gold_not_rank_1` | 529 |
| `sqlite-fts5` | `gold_rank_1` | 587 |
| `sqlite-fts5` | `miss_top_10` | 866 |
| `agentmemory` | `miss_top_10` | 1982 |

## Failure buckets

| Backend | Bucket | Questions |
| --- | --- | ---: |
| `goncho` | `missing_candidate` | 635 |
| `goncho` | `missing_companion_memory` | 203 |
| `goncho` | `rank_too_low_candidate` | 478 |
| `goncho-no-rank` | `missing_candidate` | 1960 |
| `goncho-no-rank` | `missing_companion_memory` | 3 |
| `goncho-no-rank` | `rank_too_low_candidate` | 19 |
| `bm25` | `missing_candidate` | 633 |
| `bm25` | `missing_companion_memory` | 201 |
| `bm25` | `rank_too_low_candidate` | 482 |
| `sqlite-fts5` | `missing_candidate` | 866 |
| `sqlite-fts5` | `missing_companion_memory` | 159 |
| `sqlite-fts5` | `rank_too_low_candidate` | 430 |
| `agentmemory` | `missing_candidate` | 1982 |

## Category metrics

### goncho

| Category | Questions | recall_any@5 | recall_any@10 | strict@5 | strict@10 | NDCG@5 | NDCG@10 | MRR |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `adversarial_unanswerable` | 446 | 61.66% | 71.52% | 60.09% | 69.73% | 50.59% | 53.83% | 48.90% |
| `multi_hop_retrieval` | 92 | 35.87% | 41.30% | 15.22% | 18.48% | 20.96% | 22.43% | 24.76% |
| `open_domain_retrieval` | 841 | 63.73% | 70.39% | 60.76% | 67.66% | 52.16% | 54.41% | 50.40% |
| `single_hop_retrieval` | 282 | 47.16% | 59.22% | 9.22% | 13.48% | 22.49% | 25.81% | 31.91% |
| `temporal_retrieval` | 321 | 66.98% | 71.96% | 60.75% | 65.11% | 55.15% | 56.79% | 54.47% |

### goncho-no-rank

| Category | Questions | recall_any@5 | recall_any@10 | strict@5 | strict@10 | NDCG@5 | NDCG@10 | MRR |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `adversarial_unanswerable` | 446 | 0.22% | 1.12% | 0.22% | 1.12% | 0.10% | 0.36% | 0.15% |
| `multi_hop_retrieval` | 92 | 1.09% | 2.17% | 0.00% | 1.09% | 0.18% | 0.50% | 0.38% |
| `open_domain_retrieval` | 841 | 0.24% | 1.19% | 0.24% | 1.19% | 0.10% | 0.40% | 0.18% |
| `single_hop_retrieval` | 282 | 0.35% | 0.71% | 0.00% | 0.00% | 0.22% | 0.29% | 0.41% |
| `temporal_retrieval` | 321 | 0.00% | 0.93% | 0.00% | 0.93% | 0.00% | 0.29% | 0.11% |

### bm25

| Category | Questions | recall_any@5 | recall_any@10 | strict@5 | strict@10 | NDCG@5 | NDCG@10 | MRR |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `adversarial_unanswerable` | 446 | 61.66% | 71.52% | 60.09% | 69.73% | 50.62% | 53.86% | 48.94% |
| `multi_hop_retrieval` | 92 | 35.87% | 41.30% | 15.22% | 18.48% | 20.90% | 22.37% | 24.76% |
| `open_domain_retrieval` | 841 | 63.85% | 70.39% | 60.88% | 67.78% | 52.20% | 54.42% | 50.40% |
| `single_hop_retrieval` | 282 | 46.81% | 59.22% | 9.22% | 13.48% | 22.29% | 25.70% | 31.73% |
| `temporal_retrieval` | 321 | 66.98% | 72.59% | 60.75% | 66.04% | 55.18% | 57.05% | 54.58% |

### sqlite-fts5

| Category | Questions | recall_any@5 | recall_any@10 | strict@5 | strict@10 | NDCG@5 | NDCG@10 | MRR |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `adversarial_unanswerable` | 446 | 47.53% | 52.91% | 46.19% | 51.79% | 37.35% | 39.15% | 35.29% |
| `multi_hop_retrieval` | 92 | 30.43% | 38.04% | 17.39% | 20.65% | 19.22% | 21.22% | 21.98% |
| `open_domain_retrieval` | 841 | 53.39% | 60.52% | 51.01% | 57.91% | 43.70% | 46.02% | 42.39% |
| `single_hop_retrieval` | 282 | 38.30% | 48.23% | 7.80% | 14.18% | 19.00% | 22.12% | 26.89% |
| `temporal_retrieval` | 321 | 55.76% | 62.31% | 49.84% | 56.07% | 45.48% | 47.72% | 45.65% |

### agentmemory

| Category | Questions | recall_any@5 | recall_any@10 | strict@5 | strict@10 | NDCG@5 | NDCG@10 | MRR |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `adversarial_unanswerable` | 446 | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% |
| `multi_hop_retrieval` | 92 | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% |
| `open_domain_retrieval` | 841 | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% |
| `single_hop_retrieval` | 282 | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% |
| `temporal_retrieval` | 321 | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% |

## Setup notes

- Goncho, Goncho no-rank, BM25, and SQLite FTS5 are local Go adapters with no hosted dependency.
- agentmemory probe: `python3 scripts/bench_agentmemory_locomo.py --capability`. Comparable when `AGENTMEMORY_SOURCE_DIR` points at `https://github.com/rohitg00/agentmemory` PR #583 / commit `9b18a80c9d2839b025279978d3f4b5e1f9bc6e74` with npm dependencies installed. This adapter uses the standalone InMemoryKV fallback, not the full running agentmemory server.
- mem0 probe: `python3 scripts/bench_mem0_locomo.py --capability`. Exact package version used here: none; backend is marked not comparable before scoring. Candidate install: `pip install mem0ai` plus upstream local vector-store dependencies. Comparable only after configured local retrieval can return caller-supplied `memory_id` without answer-generation scoring.

## Interpretation

Backends marked not comparable are excluded from score claims until they implement the `MemoryBackend` contract and return the same stable `memory_id` values that were inserted. This keeps the arena fair and prevents answer-generation or LLM-judge effects from leaking into retrieval metrics.

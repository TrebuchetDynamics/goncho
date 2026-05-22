# LOCOMO smoke Retrieval Report

LOCOMO smoke validates the benchmark harness. It is not a publishable full benchmark result.

This evaluates retrieval, not answer generation. It uses deterministic ID-based scoring and no LLM judge. `answer_hint` fields are never indexed or scored.

- JSON evidence: `./docs/benchmarks/results/locomo-smoke-goncho.json`
- Failure JSONL: `./docs/benchmarks/failures/locomo-smoke-categories.jsonl`
- Memories fixture: `./cmd/goncho-bench/testdata/locomo-smoke/memories.jsonl`
- Questions fixture: `./cmd/goncho-bench/testdata/locomo-smoke/questions.jsonl`
- Questions: `8`
- Memories: `17`
- Memory token estimate: `143`
- Database size bytes: `7314`
- Mode: `retrieval`
- Top-K: `10`
- no_llm_judge: `true`
- Reproduce: `go run ./cmd/goncho-bench --locomo-memories ./cmd/goncho-bench/testdata/locomo-smoke/memories.jsonl --locomo-questions ./cmd/goncho-bench/testdata/locomo-smoke/questions.jsonl --out ./docs/benchmarks/results/locomo-smoke-goncho.json --failures ./docs/benchmarks/failures/locomo-smoke-categories.jsonl --locomo-md-out ./docs/benchmarks/locomo-smoke.md --limit 10`

## Systems

| System | recall_any@5 | recall_any@10 | strict_recall@5 | strict_recall@10 | NDCG@5 | NDCG@10 | MRR | Search latency ms | Latency min ms | Latency p50 ms | Latency p95 ms | Latency max ms | RSS bytes |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| random | 87.50% | 100.00% | 87.50% | 100.00% | 59.66% | 63.27% | 50.63% | 0 | 0 | 0 | 0 | 0 | 12278024 |
| goncho-no-rank | 75.00% | 100.00% | 75.00% | 100.00% | 52.64% | 60.20% | 46.98% | 0 | 0 | 0 | 0 | 0 | 12278024 |
| recency | 75.00% | 100.00% | 75.00% | 100.00% | 52.64% | 60.20% | 46.98% | 0 | 0 | 0 | 0 | 0 | 12278024 |
| bm25 | 100.00% | 100.00% | 100.00% | 100.00% | 87.27% | 87.27% | 85.42% | 0 | 0 | 0 | 0 | 0 | 12278024 |
| sqlite-fts5 | 87.50% | 100.00% | 87.50% | 100.00% | 75.63% | 80.08% | 75.00% | 235 | 27 | 28 | 32 | 32 | 13916424 |
| goncho | 100.00% | 100.00% | 100.00% | 100.00% | 87.60% | 87.60% | 85.42% | 1 | 0 | 0 | 0 | 0 | 13916424 |

## Failure categories

| System | Category | Questions |
| --- | --- | ---: |
| random | `gold_not_rank_1` | 6 |
| random | `gold_rank_1` | 2 |
| goncho-no-rank | `gold_not_rank_1` | 6 |
| goncho-no-rank | `gold_rank_1` | 2 |
| recency | `gold_not_rank_1` | 6 |
| recency | `gold_rank_1` | 2 |
| bm25 | `gold_not_rank_1` | 2 |
| bm25 | `gold_rank_1` | 6 |
| sqlite-fts5 | `gold_not_rank_1` | 3 |
| sqlite-fts5 | `gold_rank_1` | 5 |
| goncho | `gold_not_rank_1` | 2 |
| goncho | `gold_rank_1` | 6 |

## Category metrics

### random

| Category | Questions | recall_any@5 | recall_any@10 | strict_recall@5 | strict_recall@10 | NDCG@5 | NDCG@10 | MRR |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `contradiction_supersession` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 43.07% | 43.07% | 25.00% |
| `gold_ambiguity` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `historical_retrieval` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 38.69% | 38.69% | 20.00% |
| `latest_state_retrieval` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 63.09% | 63.09% | 50.00% |
| `lexical_miss` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 69.34% | 69.34% | 50.00% |
| `multi_session_continuity` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `speaker_attribution` | 1 | 0.00% | 100.00% | 0.00% | 100.00% | 0.00% | 28.91% | 10.00% |
| `true_retrieval_failure` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 63.09% | 63.09% | 50.00% |

### goncho-no-rank

| Category | Questions | recall_any@5 | recall_any@10 | strict_recall@5 | strict_recall@10 | NDCG@5 | NDCG@10 | MRR |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `contradiction_supersession` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 38.69% | 38.69% | 20.00% |
| `gold_ambiguity` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `historical_retrieval` | 1 | 0.00% | 100.00% | 0.00% | 100.00% | 0.00% | 28.91% | 10.00% |
| `latest_state_retrieval` | 1 | 0.00% | 100.00% | 0.00% | 100.00% | 0.00% | 31.55% | 12.50% |
| `lexical_miss` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 69.34% | 69.34% | 50.00% |
| `multi_session_continuity` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `speaker_attribution` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 50.00% | 50.00% | 33.33% |
| `true_retrieval_failure` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 63.09% | 63.09% | 50.00% |

### recency

| Category | Questions | recall_any@5 | recall_any@10 | strict_recall@5 | strict_recall@10 | NDCG@5 | NDCG@10 | MRR |
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

| Category | Questions | recall_any@5 | recall_any@10 | strict_recall@5 | strict_recall@10 | NDCG@5 | NDCG@10 | MRR |
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

| Category | Questions | recall_any@5 | recall_any@10 | strict_recall@5 | strict_recall@10 | NDCG@5 | NDCG@10 | MRR |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `contradiction_supersession` | 1 | 0.00% | 100.00% | 0.00% | 100.00% | 0.00% | 35.62% | 16.67% |
| `gold_ambiguity` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `historical_retrieval` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 50.00% | 50.00% | 33.33% |
| `latest_state_retrieval` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `lexical_miss` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `multi_session_continuity` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 91.97% | 91.97% | 100.00% |
| `speaker_attribution` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 63.09% | 63.09% | 50.00% |
| `true_retrieval_failure` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |

### goncho

| Category | Questions | recall_any@5 | recall_any@10 | strict_recall@5 | strict_recall@10 | NDCG@5 | NDCG@10 | MRR |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `contradiction_supersession` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 63.09% | 63.09% | 50.00% |
| `gold_ambiguity` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `historical_retrieval` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 50.00% | 50.00% | 33.33% |
| `latest_state_retrieval` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `lexical_miss` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `multi_session_continuity` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 87.72% | 87.72% | 100.00% |
| `speaker_attribution` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `true_retrieval_failure` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |

## Leakage checks

- Answer text present in memory content: `12`
- Gold IDs present in memory content: `0`
- Question text present in memory content: `0`

`answer_hint` is not indexed or scored. Answer-text presence is reported because LOCOMO answers may be literal spans from the gold memories.

## Notes

- Retrieval-first only.
- No answer generation.
- No LLM judge.
- Baselines included: random, Goncho no-rank, recency, BM25, SQLite FTS5, Goncho current.
- The smoke fixture intentionally includes latest-state, historical, speaker-attribution, contradiction/supersession, multi-session, lexical miss, gold ambiguity, and true retrieval failure categories.

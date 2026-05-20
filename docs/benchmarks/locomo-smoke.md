# LOCOMO Smoke Retrieval Report

LOCOMO smoke validates the benchmark harness. It is not a publishable full benchmark result.

This evaluates retrieval, not answer generation. It uses deterministic ID-based scoring and no LLM judge. `answer_hint` fields are never indexed or scored.

- JSON evidence: `./docs/benchmarks/results/locomo-smoke-goncho.json`
- Failure JSONL: `./docs/benchmarks/failures/locomo-smoke-categories.jsonl`
- Memories fixture: `./cmd/goncho-bench/testdata/locomo-smoke/memories.jsonl`
- Questions fixture: `./cmd/goncho-bench/testdata/locomo-smoke/questions.jsonl`
- Questions: `8`
- Memories: `17`
- Mode: `retrieval`
- no_llm_judge: `true`

## Systems

| System | recall_any@5 | recall_any@10 | strict_recall@5 | strict_recall@10 | MRR |
| --- | ---: | ---: | ---: | ---: | ---: |
| random | 87.50% | 100.00% | 87.50% | 100.00% | 50.63% |
| recency | 75.00% | 100.00% | 75.00% | 100.00% | 46.98% |
| bm25 | 100.00% | 100.00% | 100.00% | 100.00% | 85.42% |
| sqlite-fts5 | 87.50% | 100.00% | 87.50% | 100.00% | 75.00% |
| goncho | 100.00% | 100.00% | 100.00% | 100.00% | 85.42% |

## Category metrics

### random

| Category | Questions | recall_any@5 | recall_any@10 | strict_recall@5 | strict_recall@10 | MRR |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `contradiction_supersession` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 25.00% |
| `gold_ambiguity` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `historical_retrieval` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 20.00% |
| `latest_state_retrieval` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 50.00% |
| `lexical_miss` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 50.00% |
| `multi_session_continuity` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `speaker_attribution` | 1 | 0.00% | 100.00% | 0.00% | 100.00% | 10.00% |
| `true_retrieval_failure` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 50.00% |

### recency

| Category | Questions | recall_any@5 | recall_any@10 | strict_recall@5 | strict_recall@10 | MRR |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `contradiction_supersession` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 20.00% |
| `gold_ambiguity` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `historical_retrieval` | 1 | 0.00% | 100.00% | 0.00% | 100.00% | 10.00% |
| `latest_state_retrieval` | 1 | 0.00% | 100.00% | 0.00% | 100.00% | 12.50% |
| `lexical_miss` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 50.00% |
| `multi_session_continuity` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `speaker_attribution` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 33.33% |
| `true_retrieval_failure` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 50.00% |

### bm25

| Category | Questions | recall_any@5 | recall_any@10 | strict_recall@5 | strict_recall@10 | MRR |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `contradiction_supersession` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 50.00% |
| `gold_ambiguity` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `historical_retrieval` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 33.33% |
| `latest_state_retrieval` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `lexical_miss` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `multi_session_continuity` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `speaker_attribution` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `true_retrieval_failure` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |

### sqlite-fts5

| Category | Questions | recall_any@5 | recall_any@10 | strict_recall@5 | strict_recall@10 | MRR |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `contradiction_supersession` | 1 | 0.00% | 100.00% | 0.00% | 100.00% | 16.67% |
| `gold_ambiguity` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `historical_retrieval` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 33.33% |
| `latest_state_retrieval` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `lexical_miss` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `multi_session_continuity` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `speaker_attribution` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 50.00% |
| `true_retrieval_failure` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |

### goncho

| Category | Questions | recall_any@5 | recall_any@10 | strict_recall@5 | strict_recall@10 | MRR |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `contradiction_supersession` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 50.00% |
| `gold_ambiguity` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `historical_retrieval` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 33.33% |
| `latest_state_retrieval` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `lexical_miss` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `multi_session_continuity` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `speaker_attribution` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |
| `true_retrieval_failure` | 1 | 100.00% | 100.00% | 100.00% | 100.00% | 100.00% |

## Notes

- Retrieval-first only.
- No answer generation.
- No LLM judge.
- Baselines included: random, recency, BM25, SQLite FTS5, Goncho.
- The smoke fixture intentionally includes latest-state, historical, speaker-attribution, contradiction/supersession, multi-session, lexical miss, gold ambiguity, and true retrieval failure categories.

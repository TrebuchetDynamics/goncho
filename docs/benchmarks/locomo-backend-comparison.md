# LOCOMO External Backend Comparison

This is a benchmark adapter suite, not a marketing dunk. It compares retrieval backends only when they can return stable inserted memory IDs.

- JSON evidence: `docs/benchmarks/results/locomo-backend-comparison.json`
- Source report: `docs/benchmarks/results/locomo-2026-05-20-goncho.json`
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
- publish failure categories, not just scores

## Results

| Backend | Comparable | recall_any@5 | recall_any@10 | strict@5 | strict@10 | MRR | Notes |
| --- | --- | ---: | ---: | ---: | ---: | ---: | --- |
| `goncho` | true | 52.47% | 58.73% | 44.80% | 49.95% | 41.04% | Scores imported from committed pinned deterministic LOCOMO harness report; adapter interface added in cmd/goncho-bench. |
| `agentmemory` | false | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | not comparable yet: local reference exposes MCP/REST product surfaces, but no stable-memory-id adapter is wired to return inserted LOCOMO memory_id values |
| `mem0` | false | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | not comparable yet: no stable-memory-id adapter is wired for this backend; retrieval must return original inserted LOCOMO memory_id values before scoring |
| `memo0` | false | 0.00% | 0.00% | 0.00% | 0.00% | 0.00% | not comparable yet: no stable-memory-id adapter is wired for this backend; retrieval must return original inserted LOCOMO memory_id values before scoring |
| `bm25` | true | 60.19% | 67.96% | 51.26% | 57.87% | 46.88% | Scores imported from committed pinned deterministic LOCOMO harness report; adapter interface added in cmd/goncho-bench. |
| `sqlite-fts5` | true | 49.14% | 56.76% | 42.13% | 48.54% | 38.16% | Scores imported from committed pinned deterministic LOCOMO harness report; adapter interface added in cmd/goncho-bench. |
| `recency` | true | 0.40% | 0.81% | 0.30% | 0.55% | 0.22% | Scores imported from committed pinned deterministic LOCOMO harness report; adapter interface added in cmd/goncho-bench. |
| `random` | true | 1.31% | 2.47% | 0.86% | 1.82% | 0.80% | Scores imported from committed pinned deterministic LOCOMO harness report; adapter interface added in cmd/goncho-bench. |

## Failure categories

For comparable backends, failure categories are deterministic retrieval buckets from the same gold IDs:

- `gold_rank_1`: first retrieved memory is gold.
- `gold_not_rank_1`: gold appears below rank 1.
- `miss_top_10`: no gold memory appears in top 10.

| Backend | gold_rank_1 | gold_not_rank_1 | miss_top_10 |
| --- | ---: | ---: | ---: |
| `goncho` | 651 | 513 | 818 |
| `bm25` | 733 | 614 | 635 |
| `sqlite-fts5` | 584 | 541 | 857 |
| `recency` | 2 | 14 | 1966 |
| `random` | 6 | 43 | 1933 |

## Adapter contract

Each external backend must implement:

```text
MemoryBackend:
  Reset()
  Insert(memory_id, content, metadata)
  Search(question, topK) -> []Result{memory_id, score}
```

If a backend cannot return the exact inserted `memory_id`, it is marked `not comparable` and excluded from score claims.

## Setup notes

- `agentmemory`: needs a stable-ID adapter over its local server/MCP/REST surface before scores can be published.
- `mem0` / `memo0`: needs a stable-ID adapter that preserves caller-supplied IDs through search results.
- Built-in baselines and Goncho are currently sourced from the committed pinned deterministic LOCOMO report while the adapter suite is being wired for full external runs.

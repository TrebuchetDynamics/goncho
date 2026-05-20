---
title: Benchmark Roadmap
description: Long-term evaluation plan for Goncho as an agent memory system.
---

Goncho is being evaluated as a long-term memory retrieval system for agents, not just a vector store.

LongMemEval-S is the first layer: deterministic retrieval sanity on long conversational haystacks. The next benchmarks should stress progressively harder memory behavior: evolving facts, temporal state, scale, noise, standard IR credibility, and real agent utility.

## Progression

| Phase | Benchmark | Purpose | Status |
| --- | --- | --- | --- |
| 1 | LongMemEval-S | Long conversational retrieval with ID-based evidence scoring. | First scientific pass done. |
| 2 | LOCOMO | Conversational long-term memory: evolving facts, temporal recall, contradictions. | Next target. |
| 3 | InfiniteBench / RULER | Scale, buried-fact, distractor, and long-context stress. | Planned. |
| 4 | BABILong | Controlled synthetic temporal/entity tracking. | Planned. |
| 5 | BEIR | Standard IR credibility against established retrieval baselines. | Planned. |
| 6 | Real-world replay | Actual agent utility on real sessions, mistakes, preferences, and handoffs. | Planned. |

## Best next target: LOCOMO

LOCOMO is the best next benchmark because it is closer to real agent memory than plain retrieval.

It tests:

- long conversations,
- temporal memory,
- multi-session recall,
- evolving facts,
- contradictions over time,
- relationship and event changes.

For Goncho, LOCOMO should answer:

- Does Goncho preserve older truth while surfacing current truth?
- Does it handle changed preferences and relationships?
- Can it avoid presenting stale facts as current facts?
- Can review/staleness warnings explain uncertainty?

## Scale stress: InfiniteBench and RULER

These benchmarks should test:

- retrieval with huge memory pools,
- buried facts,
- distractors,
- structured retrieval,
- memory growth degradation,
- context budget pressure.

Report scaling curves, not only final recall.

## Controlled science: BABILong

BABILong is synthetic but useful for controlled checks:

- temporal reasoning,
- entity tracking,
- simple multi-hop consistency,
- repeated facts under distractors.

It should not be the only benchmark, but it can isolate specific failure modes.

## IR credibility: BEIR

BEIR is not agent-memory-specific, but it matters. Goncho should be compared against:

- random,
- BM25,
- SQLite FTS5,
- vector-only,
- hybrid BM25+vector where available.

If Goncho cannot compete with standard retrieval systems, agent-memory claims become weaker.

## Most important eventual target: real-world replay

Synthetic benchmarks only go so far. Goncho eventually needs real agent session replay:

- real coding sessions,
- real chat preferences,
- rejected approaches,
- stale code paths,
- recurring mistakes,
- user corrections,
- handoffs and compactions.

Example checks:

- “Three days ago the user rejected Redis. Did Goncho remember that?”
- “The file moved after memory was written. Did Goncho verify live state before trusting it?”
- “The agent repeated a failed Docker fix before. Did Goncho warn?”
- “A prompt-injection import entered memory. Did Goncho quarantine it?”

## Metrics to report

Do not report only recall. Serious Goncho evaluations should include:

- recall@K,
- recall_any@K when the benchmark uses any-gold-session scoring,
- MRR,
- NDCG where applicable,
- latency min/p50/p95/max,
- RSS / peak memory,
- database size,
- memory count and total token estimate,
- degradation as distractors increase,
- stale-memory warning rate,
- contradiction handling accuracy,
- leakage counts,
- failure categories.

## Scientific controls

Every benchmark should include:

1. Pinned dataset source and revision.
2. Raw artifact checksum.
3. Conversion script.
4. Converted artifact checksum when practical.
5. Deterministic scoring by evidence ID, not LLM judgment, unless explicitly running answer generation.
6. Baselines: random, BM25, SQLite FTS5, Goncho without current ranking, Goncho current.
7. Leakage checks for query text, gold IDs, and answer labels.
8. Failure audit with top-10 retrieval and miss reason.
9. One-command clean-room reproduction where licensing permits.
10. CI-safe smoke target with tiny pinned fixtures.

See also: [`docs/benchmarks/ROADMAP.md`](../../../docs/benchmarks/ROADMAP.md).

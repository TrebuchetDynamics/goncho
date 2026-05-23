---
title: Benchmark Roadmap
description: Long-term evaluation plan for Goncho as an agent memory system.
---

Goncho is being evaluated as a long-term memory retrieval system for agents, not just a vector store.

LongMemEval-S is the first layer: deterministic retrieval sanity on long conversational haystacks. The next benchmarks should stress progressively harder memory behavior: evolving facts, temporal state, scale, noise, standard IR credibility, and real agent utility.

For runnable commands, current frozen artifacts, and external-backend adapter status, see [Retrieval Benchmarks](/reference/retrieval-benchmarks/). This roadmap stays focused on the longer evaluation sequence.

## Progression

| Phase | Benchmark | Purpose | Status |
| --- | --- | --- | --- |
| 1 | LongMemEval-S | Long conversational retrieval with ID-based evidence scoring. | First scientific pass done. |
| 2 | LOCOMO | Conversational long-term memory: evolving facts, temporal recall, contradictions. | Candidate-generation milestone and stable-ID backend comparison frozen. |
| 3 | InfiniteBench / RULER | Scale, buried-fact, distractor, and long-context stress. | Planned. |
| 4 | BABILong | Controlled synthetic temporal/entity tracking. | Planned. |
| 5 | BEIR | Standard IR credibility against established retrieval baselines. | Planned. |
| 6 | Real-world replay | Actual agent utility on real sessions, mistakes, preferences, and handoffs. | Planned. |

## LOCOMO status

LOCOMO is the best benchmark after LongMemEval-S because it is closer to real agent memory than plain retrieval.

Current milestone:

- Goncho recall_any@5 improved `0.5247 -> 0.6014`.
- Goncho recall_any@10 improved `0.5873 -> 0.6791`.
- Goncho MRR improved `0.4104 -> 0.4690`.
- BM25-win `missing_candidate` failures dropped `164 -> 2`.
- LongMemEval-S stayed stable.

The lesson: LOCOMO was not improved by clever reranking. It exposed a candidate-generation weakness, and the fix widened lexical pre-rank candidates before top-K truncation.

The backend harness now compares Goncho, BM25, SQLite FTS5, agentmemory, and mem0 with the same LOCOMO data and centralized scoring. Backends that cannot return stable inserted memory IDs are marked `not comparable`. Preserve the frozen comparison artifacts before adding contradiction/staleness audits or making additional external backends comparable.

LOCOMO improvement priorities:

- Use multi-hop graph expansion to connect entities, events, relationships, and evidence IDs that lexical matching alone cannot bridge.
- Add query decomposition so multi-part questions retrieve each required fact before final ranking.
- Add coverage-aware ranking so top results include complementary gold memories instead of near-duplicate hits.
- Improve temporal and speaker routing so changed facts, chronology, and who-said-what are ranked in the right conversation branch.
- Drive changes from failure-audit buckets such as missing candidates, rank-too-low candidates, wrong branch retrieval, and missing companion memories.
- Target: raise multi-hop recall_any@10 above `50%` and raise multi-hop strict_recall@10 above `25%` without answer hints, benchmark-specific hacks, or LLM judges.

LOCOMO implementation gate:

- Recommendations are not approval to change retrieval behavior.
- Write an approved design or plan before production retrieval changes.
- Start implementation with a focused failing recall test, for example `TestGraphRecallConnectsOwnerThroughServiceRelation`, before adding graph storage, relation extraction, or reranking code.
- Preserve frozen LOCOMO artifacts until a new date-stamped full run is intentionally generated.
- Do not tune against LOCOMO gold IDs, answer text, or benchmark-specific hacks; score only stable inserted memory IDs.

First graph-assisted implementation slice delivered: `TestGraphRecallConnectsOwnerThroughServiceRelation` proves graph-expanded multi-hop recall can retrieve a stable-ID companion memory with relation path provenance before any LOCOMO full-run artifact is regenerated.

Coverage-aware graph companion selection delivered: `TestRecallPipelineCoverageAwareSelectionKeepsGraphCompanion` proves selection prefers relation-path companion memories over near-duplicate lexical hits without regenerating LOCOMO full-run artifacts.

Query-decomposition recall slice delivered: `TestRecallQueryDecompositionRetrievesEachSubQuestionFact` proves multi-part questions can split into subqueries, retrieve each required stable-ID fact, and merge results before scoring without regenerating LOCOMO full-run artifacts.

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

## External backend credibility

External comparisons must keep the harness more trustworthy than the backend.

Rules:

- adapters stay isolated,
- scoring stays centralized,
- every backend uses the same JSONL,
- every backend uses the same gold IDs,
- every backend gets the same metrics,
- every backend gets the same leakage checks,
- every backend gets the same failure taxonomy,
- no adapter may rewrite scoring semantics,
- no content-only matching unless collision-safe.

Current LOCOMO backend status:

| Backend | Comparable | Notes |
| --- | --- | --- |
| Goncho | yes | Local deterministic adapter. |
| BM25 | yes | Local lexical baseline. |
| SQLite FTS5 | yes | Local SQLite FTS baseline. |
| agentmemory | yes, PR standalone fallback | PR #583 stable IDs work; standalone InMemoryKV fallback scored `0.0000` on LOCOMO full and is not the full running server. |
| mem0 | no | Package not installed locally; no stable-ID run exists. |

See `docs/benchmarks/external-backend-adapters.md` for the adapter contract and operator notes.

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

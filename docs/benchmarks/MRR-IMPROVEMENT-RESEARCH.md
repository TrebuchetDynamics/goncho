# MRR Improvement Research — LongMemEval-S

Date: 2026-05-20

This note researches how to increase Goncho's LongMemEval-S MRR without weakening benchmark credibility.

## Current evidence

Source report:

- `docs/benchmarks/results/longmemeval-s-2026-05-20-goncho.json`

Original pre-fix metrics that triggered this investigation:

| Metric | Value |
| --- | ---: |
| Questions | 500 |
| recall_any@5 | 96.40% |
| recall_any@10 | 98.00% |
| MRR | 81.12% |

Original pre-fix rank distribution:

| Bucket | Count |
| --- | ---: |
| Rank 1 | 343 |
| Rank 2 | 102 |
| Rank 3-5 | 37 |
| Rank 6-10 | 8 |
| Miss top 10 | 10 |

After the peer-scoped benchmark mapping fix, current regenerated metrics are:

| Metric | Value |
| --- | ---: |
| Questions | 500 |
| recall_any@5 | 96.80% |
| recall_any@10 | 98.00% |
| MRR | 91.35% |

The original MRR was mainly limited by rank-2 cases caused by cross-peer duplicate-content ID mapping. Current residual MRR is limited by a smaller set of true rank-ordering and miss cases.

## Failure categories

A quick audit of non-rank-1 cases produced these categories:

| Category | Count | MRR implication |
| --- | ---: | --- |
| Top result has same base ID as gold | 87 | Biggest lever; many rank-2 cases are near-duplicate or paired raw/answer sessions. |
| Numeric / temporal ordering | 22 | Needs date/numeric reasoning features, not just lexical overlap. |
| Other lexical/semantic confusion | 22 | Needs better query expansion, role weighting, or embeddings. |
| Preference / follow-up recommendation | 11 | Needs preference-aware and assistant-answer-aware retrieval. |
| Miss top 10 | 10 | Needs recall expansion, hybrid retrieval, or conversion review. |
| Abstention variant confusion | 5 | Needs explicit handling of answerable vs abstention variants. |

Potential impact estimates:

| Hypothesis | Approximate MRR if fully solved |
| --- | ---: |
| Current | 0.8112 |
| Same-base top result promoted to rank 1 | 0.9007 |
| All rank-2 cases promoted to rank 1 | 0.9132 |

These are upper-bound estimates, not benchmark claims.

## Most important finding

2026-05-20 follow-up: the largest MRR loss was a **benchmark harness ID-mapping bug**, not a retrieval-model weakness.

The Goncho benchmark converted a `Search` hit back to LongMemEval memory IDs through a global `content -> []id` map. LongMemEval reuses identical session text across different question peers, so a hit from peer `p2` could be reported first as an ID belonging to peer `p1` when the content was identical. This preserved high recall_any@K but depressed strict ID MRR.

A peer-scoped map, `peer + content -> []id`, fixes this measurement artifact.

Measured locally on the pinned cached LongMemEval-S conversion:

| System / condition | MRR | recall_any@5 | recall_any@10 |
| --- | ---: | ---: | ---: |
| Published Goncho report before peer-scoped ID mapping | 0.8112 | 0.964 | 0.980 |
| Goncho after peer-scoped ID mapping | 0.9105 | 0.968 | 0.980 |
| Goncho after conservative temporal reranking | 0.9135 | 0.968 | 0.980 |
| Standalone BM25 baseline | 0.9105 | 0.968 | 0.980 |

New strict rank distribution after peer-scoped mapping:

| Bucket | Count |
| --- | ---: |
| Rank 1 | 435 |
| Rank 2 | 25 |
| Rank 3 | 13 |
| Rank 4 | 7 |
| Rank 5 | 4 |
| Rank 6-10 | 6 |
| Miss top 10 | 10 |

Conclusion: the first and safest way to increase reported MRR is to fix benchmark identity mapping and regenerate the benchmark report. After that, remaining MRR work should target the true residual failures: 25 rank-2 cases, 13 rank-3 cases, and 10 top-10 misses.

The highest-impact retrieval issue after the harness fix is no longer broad recall. It is **residual top-rank ordering among a much smaller set of hard cases**.

Follow-up failure classification artifacts:

- `docs/benchmarks/failures/longmemeval-s-2026-05-20-categories.jsonl`
- `docs/benchmarks/longmemeval-s-2026-05-20-failure-categories.md`

Current hard-case counts after classification:

| Bucket | Count |
| --- | ---: |
| rank-2 cases | 22 |
| rank-3 cases | 13 |
| misses in top 10 | 10 |

Conservative temporal reranking moved 3 rank-2 temporal cases to rank 1 and one rank-7 case to rank 6, with no recall_any@10 regression in the full cached LongMemEval-S run. Remaining largest category: `temporal_ambiguity` with 23 cases.

Common pattern:

```text
query: What type of action figure did I buy from a thrift store?
gold:  answer_5cc9b056
top:   5cc9b056
rank:  2
```

The top result often has the same base session identifier as the gold session but is not the exact gold ID. This means Goncho is finding the right neighborhood but not always the exact evidence item first.

## Research directions

### 1. Direct-answer/session role weighting

Goal: rank the session that contains the answer-bearing assistant response above nearby setup or raw variants.

Possible features:

- boost memories where final assistant turn contains high query-token overlap;
- boost sessions with explicit answer-like phrasing after a user question;
- for `who/what/when/how many/how much` queries, boost concise assistant turns containing named entities, dates, or numbers;
- penalize long distractor sessions when a shorter answer-bearing session has similar BM25 score.

Scientific caution:

- Do not use gold ID prefixes or `answer_` metadata as a ranking feature in production scoring.
- It is acceptable to use role/content structure because real Goncho memories preserve message roles and evidence provenance.

Expected impact:

- Strong on same-base rank-2 cases.
- Likely improves MRR more than recall_any@K.

### 2. Pair/duplicate-aware evidence grouping

Goal: recognize when multiple sessions are semantically near-duplicates or variants of the same evidence event.

Two separate options:

1. **Report-only grouping**: keep benchmark scoring strict, but add analysis showing same-base/near-duplicate misses.
2. **Retrieval grouping**: collapse variants before ranking, then choose representative evidence by answer-likeness.

Scientific caution:

- Do not change official ID scoring by treating same-base IDs as correct unless the report labels it as a separate diagnostic metric.
- Avoid dataset-specific base-ID hacks in Goncho ranking.

Expected impact:

- Explains most MRR loss.
- May enable principled deduplication in real memory, where repeated evidence should not crowd out canonical evidence.

### 3. Temporal and numeric query features

Goal: improve questions asking for counts, dates, order, duration, and recency.

Examples:

- “How many days…”
- “What is the order…”
- “Which event happened first…”
- “How long…”

Possible features:

- detect temporal/numeric query intent;
- boost memories containing dates, durations, numbers, or ordered event language;
- incorporate `haystack_dates` during benchmark conversion as non-retrievable metadata;
- support metadata-aware temporal scoring without leaking gold labels.

Scientific caution:

- Metadata dates are valid if they come from the benchmark haystack, not from gold labels.
- Date metadata should be reported as a scoring feature in JSON.

Expected impact:

- Helps 22 numeric/temporal ordering cases.
- Also supports Goncho's real product goals around stale/current temporal memory.

### 4. Abstention/answerability variant handling

Goal: avoid ranking answerable variants above abstention variants, or vice versa.

Observed pattern:

```text
gold: answer_586de428_abs
top:  answer_586de428
```

Possible features:

- detect answerability or missing-information cues;
- separate answerable vs unanswerable memories in lifecycle/status metadata;
- add abstention diagnostic metrics rather than hiding failures.

Scientific caution:

- Do not use `_abs` suffix directly as a production ranking feature.
- It can be used in benchmark analysis to classify failures.

Expected impact:

- Small count, but important for trust-preserving memory.

### 5. Hybrid lexical + local embedding retrieval

Goal: improve semantic misses where lexical overlap is weak.

Options:

- local embedding lane using a pinned local model;
- reciprocal-rank fusion between BM25-style lexical score and vector score;
- optional because Goncho's base story remains local-first and no cloud dependency.

Scientific caution:

- Pin model name, revision, runtime, and embedding dimensions.
- Add vector-only and BM25+vector baselines.
- Keep no-network reproducibility if possible.

Expected impact:

- Likely helps “preference/follow-up recommendation” and semantic confusion cases.
- May improve MRR and recall_any@10.

### 6. Query expansion and synonym normalization

Goal: improve semantic matching without embeddings.

Possible features:

- lightweight synonym table for common LongMemEval/user-memory terms;
- normalize possessives and contractions;
- improve stemming: `gave/given`, `bought/buy`, `flied/flew`, `attended/attend`;
- role-aware tokenization that weighs user preference statements and assistant answers differently.

Scientific caution:

- Keep expansion general, not dataset-specific.
- Report exact expansion rules.

Expected impact:

- Helps low-risk lexical misses.
- Less likely to move MRR as much as direct-answer weighting.

## Recommended experiment order

### Experiment A: Add failure-category report

Before changing ranking again, emit category diagnostics from the JSON report:

- same-base top result,
- abstention variant confusion,
- numeric/temporal query,
- preference/recommendation query,
- miss top 10,
- other.

Why first:

- Makes every future MRR gain explainable.
- Prevents accidental metric gaming.

### Experiment B: Direct-answer role weighting

Add content-only scoring features:

- final assistant-turn overlap,
- answer-like turn boost,
- number/date/entity presence for matching query types,
- length normalization for concise answer turns.

Acceptance:

- improves MRR on full LongMemEval-S;
- does not reduce recall_any@5 or recall_any@10 materially;
- smoke benchmarks still pass;
- failure audit count does not grow.

### Experiment C: Temporal/numeric feature lane

Add optional metadata-aware temporal scoring from haystack dates.

Acceptance:

- improves numeric/temporal category MRR;
- no leakage from answer IDs or gold labels;
- reports feature usage in generated JSON.

### Experiment D: Optional local embedding lane

Add pinned local vector baseline and hybrid fusion.

Acceptance:

- vector-only and hybrid reports generated;
- all model/version/runtime details pinned;
- no cloud dependency for the benchmark claim.

## What not to do

- Do not rank by `answer_` ID prefix.
- Do not treat same-base IDs as correct in the official metric.
- Do not hide the one query-in-memory leakage case.
- Do not use an LLM judge for retrieval accuracy.
- Do not tune only on the final score without preserving failure audits.

## Best next implementation slice

1. Fix peer-scoped content ID mapping in `cmd/goncho-bench`.
2. Regenerate the canonical LongMemEval-S Goncho report.
3. Then implement `bench-failures classify` or equivalent report-generation logic that reads the regenerated report and writes:

```text
docs/benchmarks/failures/longmemeval-s-2026-05-20-categories.json
```

After peer-scoped mapping, use that category file to target direct-answer role weighting and temporal/numeric reranking.

This keeps the work scientific: fix measurement first, understand residual rank loss second, change retrieval ranking third.

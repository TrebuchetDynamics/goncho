# MRR Improvement Research — LongMemEval-S

Date: 2026-05-20

This note researches how to increase Goncho's LongMemEval-S MRR without weakening benchmark credibility.

## Current evidence

Source report:

- `docs/benchmarks/results/longmemeval-s-2026-05-20-goncho.json`

Current metrics:

| Metric | Value |
| --- | ---: |
| Questions | 500 |
| recall_any@5 | 96.40% |
| recall_any@10 | 98.00% |
| MRR | 81.12% |

Rank distribution:

| Bucket | Count |
| --- | ---: |
| Rank 1 | 343 |
| Rank 2 | 102 |
| Rank 3-5 | 37 |
| Rank 6-10 | 8 |
| Miss top 10 | 10 |

MRR is mainly limited by rank-2 cases, not top-10 recall. Moving rank-2 cases to rank 1 is the largest near-term lever.

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

The highest-impact issue is not recall. It is **tie-breaking and near-duplicate ordering**.

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

Implement `bench-failures classify` or equivalent report-generation logic that reads `docs/benchmarks/results/longmemeval-s-2026-05-20-goncho.json` and writes:

```text
docs/benchmarks/failures/longmemeval-s-2026-05-20-categories.json
```

Then use that category file to target direct-answer role weighting.

This keeps the work scientific: understand rank loss first, change ranking second.

# LOCOMO Candidate-Generation Milestone — 2026-05-20

This milestone freezes the first LOCOMO-driven optimization result for Goncho.

## Finding

LOCOMO exposed a candidate-generation weakness in Goncho. The problem was not primarily ranking philosophy: BM25 was winning mostly because Goncho excluded many gold memories before ranking.

The fix widened lexical pre-rank candidate generation so LOCOMO-scale conversations are ranked before top-K truncation.

## Result

| Metric | Before | After |
| --- | ---: | ---: |
| LOCOMO Goncho recall_any@5 | `0.5247` | `0.6014` |
| LOCOMO Goncho recall_any@10 | `0.5873` | `0.6791` |
| LOCOMO Goncho MRR | `0.4104` | `0.4690` |
| BM25-win `missing_candidate` failures | `164` | `2` |

After the change, Goncho essentially matches BM25 on full LOCOMO retrieval while preserving LongMemEval-S performance.

## Controls preserved

- No LLM judge.
- No answer-generation scoring.
- No benchmark-specific gold-ID hack.
- No ranking change.
- No gold leakage.
- Same LOCOMO JSONL conversion and ID-based scoring.
- Same leakage checks and failure taxonomy.

## LongMemEval-S preservation

LongMemEval-S remained stable after the candidate-generation change:

| Metric | After |
| --- | ---: |
| recall_any@5 | `0.968` |
| recall_any@10 | `0.980` |
| MRR | `0.9135` |

## Interpretation

LOCOMO was not solved by clever reranking. It was improved by fixing candidate generation.

This is the benchmark lesson to preserve before further tuning: a retrieval system cannot rank evidence it never admits into the candidate set.

## Evidence

- Full LOCOMO report: [`docs/benchmarks/locomo-2026-05-20.md`](locomo-2026-05-20.md)
- BM25 vs Goncho failure analysis: [`docs/benchmarks/locomo-2026-05-20-failure-analysis.md`](locomo-2026-05-20-failure-analysis.md)
- Comparison JSONL: [`docs/benchmarks/failures/locomo-2026-05-20-bm25-vs-goncho.jsonl`](failures/locomo-2026-05-20-bm25-vs-goncho.jsonl)
- LOCOMO result JSON: [`docs/benchmarks/results/locomo-2026-05-20-goncho.json`](results/locomo-2026-05-20-goncho.json)
- LongMemEval-S result JSON: [`docs/benchmarks/results/longmemeval-s-2026-05-20-goncho.json`](results/longmemeval-s-2026-05-20-goncho.json)

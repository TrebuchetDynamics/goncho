# LOCOMO BM25 vs Goncho Failure Analysis — 2026-05-20

Diagnosis only: no ranking changes, no LLM judge, no answer-generation scoring.

- Source report: `docs/benchmarks/results/locomo-2026-05-20-goncho.json`
- JSONL comparison: `docs/benchmarks/failures/locomo-2026-05-20-bm25-vs-goncho.jsonl`
- Questions: `1982`

## Summary metrics

| System | recall_any@5 | recall_any@10 | strict@5 | strict@10 | MRR |
| --- | ---: | ---: | ---: | ---: | ---: |
| bm25 | 60.19% | 67.96% | 51.26% | 57.87% | 46.88% |
| goncho | 60.14% | 67.91% | 51.16% | 57.67% | 46.90% |

## Candidate-generation slice result

Before widening lexical candidate generation, BM25 led Goncho because many gold memories were excluded before ranking:

| Metric | Before | After |
| --- | ---: | ---: |
| Goncho recall_any@5 | 52.47% | 60.14% |
| Goncho recall_any@10 | 58.73% | 67.91% |
| Goncho MRR | 41.04% | 46.90% |
| BM25 wins over Goncho | 257 | 18 |
| Goncho wins over BM25 | 162 | 14 |
| Ties | 953 | 1316 |
| Both miss | 610 | 634 |
| BM25-win `missing_candidate` | 164 | 2 |

LongMemEval-S stayed stable after the candidate-generation change: recall_any@5 `0.968`, recall_any@10 `0.980`, MRR `0.9135`.

## Winner counts

| Winner | Count |
| --- | ---: |
| BM25 wins | 18 |
| Goncho wins | 14 |
| Ties | 1316 |
| Both miss | 634 |

## Category breakdown

| Category | BM25 wins | Goncho wins | Ties | Both miss |
| --- | ---: | ---: | ---: | ---: |
| `adversarial_unanswerable` | 4 | 2 | 313 | 127 |
| `multi_hop_retrieval` | 1 | 1 | 36 | 54 |
| `open_domain_retrieval` | 3 | 4 | 584 | 250 |
| `single_hop_retrieval` | 6 | 5 | 156 | 115 |
| `temporal_retrieval` | 4 | 2 | 227 | 88 |

## BM25-win failure modes

| Failure mode | Count |
| --- | ---: |
| `gold_ambiguity` | 8 |
| `missing_candidate` | 2 |
| `rerank_regression` | 8 |

## Worst BM25-over-Goncho cases

- `locomo-conv-50-q-001` delta `999991` mode `missing_candidate`: When did Calvin first travel to Tokyo?
- `locomo-conv-41-q-028` delta `999989` mode `missing_candidate`: When did John have a party with veterans?
- `locomo-conv-42-q-017` delta `2` mode `gold_ambiguity`: What physical transformation did Nate undergo in April 2022?
- `locomo-conv-26-q-041` delta `1` mode `gold_ambiguity`: How many times has Melanie gone to the beach in 2023?
- `locomo-conv-26-q-052` delta `1` mode `gold_ambiguity`: What has Melanie painted?
- `locomo-conv-30-q-049` delta `1` mode `rerank_regression`: What did Gina design for her store?
- `locomo-conv-30-q-089` delta `1` mode `rerank_regression`: What did Jon design for his store?
- `locomo-conv-41-q-048` delta `1` mode `gold_ambiguity`: What exercises has John done?
- `locomo-conv-41-q-182` delta `1` mode `rerank_regression`: What did John take away from visiting the orphanage?
- `locomo-conv-42-q-002` delta `1` mode `gold_ambiguity`: What kind of interests do Joanna and Nate share?

## Top Goncho-over-BM25 cases

- `locomo-conv-50-q-021` delta `-999989` mode `gold_ambiguity`: Who inspired Dave's passion for car engineering?
- `locomo-conv-26-q-057` delta `-2` mode `gold_ambiguity`: What symbols are important to Caroline?
- `locomo-conv-42-q-060` delta `-2` mode `gold_ambiguity`: What does Joanna do to remember happy memories?
- `locomo-conv-26-q-004` delta `-1` mode `entity_exactness`: What did Caroline research?
- `locomo-conv-26-q-050` delta `-1` mode `temporal_evolution`: When did Caroline and Melanie go to a pride fesetival together?
- `locomo-conv-42-q-085` delta `-1` mode `gold_ambiguity`: Was the first half of September 2022 a good month career-wise for Nate and Joanna? Answer yes or no.
- `locomo-conv-42-q-096` delta `-1` mode `gold_ambiguity`: What is Nate's favorite video game?
- `locomo-conv-42-q-123` delta `-1` mode `entity_exactness`: What did Nate do for Joanna on 25 May, 2022?
- `locomo-conv-43-q-152` delta `-1` mode `gold_ambiguity`: What is the topic of discussion between John and Tim on 11 December, 2023?
- `locomo-conv-47-q-177` delta `-1` mode `entity_exactness`: Where does John get his ideas from?

## Interpretation

The original BM25 lead was primarily candidate generation: Goncho searched too small a pre-rank conclusion window for LOCOMO-scale conversations. After widening lexical candidate generation before top-K truncation, `missing_candidate` nearly disappeared (`164 -> 2`) and Goncho now essentially matches BM25 overall.

Remaining BM25 wins are now mostly `gold_ambiguity` and `rerank_regression`, not broad candidate loss. Recommended next slice: inspect the remaining 18 BM25-win rows manually before any further ranking changes.

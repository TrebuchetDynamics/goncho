# LOCOMO BM25 vs Goncho Failure Analysis — 2026-05-20

Diagnosis only: no ranking changes, no LLM judge, no answer-generation scoring.

- Source report: `docs/benchmarks/results/locomo-2026-05-20-goncho.json`
- JSONL comparison: `docs/benchmarks/failures/locomo-2026-05-20-bm25-vs-goncho.jsonl`
- Questions: `1982`

## Summary metrics

| System | recall_any@5 | recall_any@10 | strict@5 | strict@10 | MRR |
| --- | ---: | ---: | ---: | ---: | ---: |
| bm25 | 60.19% | 67.96% | 51.26% | 57.87% | 46.88% |
| goncho | 52.47% | 58.73% | 44.80% | 49.95% | 41.04% |

## Winner counts

| Winner | Count |
| --- | ---: |
| BM25 wins | 257 |
| Goncho wins | 162 |
| Ties | 953 |
| Both miss | 610 |

## Category breakdown

| Category | BM25 wins | Goncho wins | Ties | Both miss |
| --- | ---: | ---: | ---: | ---: |
| `adversarial_unanswerable` | 55 | 38 | 229 | 124 |
| `multi_hop_retrieval` | 9 | 7 | 22 | 54 |
| `open_domain_retrieval` | 98 | 73 | 434 | 236 |
| `single_hop_retrieval` | 44 | 25 | 101 | 112 |
| `temporal_retrieval` | 51 | 19 | 167 | 84 |

## BM25-win failure modes

| Failure mode | Count |
| --- | ---: |
| `gold_ambiguity` | 61 |
| `missing_candidate` | 164 |
| `rerank_regression` | 32 |

## Worst BM25-over-Goncho cases

- `locomo-conv-41-q-005` delta `999998` mode `missing_candidate`: When did John join the online support group?
- `locomo-conv-41-q-012` delta `999998` mode `gold_ambiguity`: What people has Maria met and helped while volunteering?
- `locomo-conv-41-q-014` delta `999998` mode `missing_candidate`: When did Maria's grandmother pass away?
- `locomo-conv-41-q-069` delta `999998` mode `missing_candidate`: What type of workout class did Maria start doing in December 2023?
- `locomo-conv-41-q-071` delta `999998` mode `missing_candidate`: What kind of meal did John and his family make together in the photo shared by John?
- `locomo-conv-41-q-072` delta `999998` mode `missing_candidate`: What kind of online group did John join?
- `locomo-conv-41-q-074` delta `999998` mode `missing_candidate`: Who inspired Maria to start volunteering?
- `locomo-conv-41-q-078` delta `999998` mode `missing_candidate`: What activity did John's colleague, Rob, invite him to?
- `locomo-conv-41-q-079` delta `999998` mode `missing_candidate`: What is the name of John's one-year-old child?
- `locomo-conv-41-q-081` delta `999998` mode `missing_candidate`: What did Maria make for her home to remind her of a trip to England?

## Top Goncho-over-BM25 cases

- `locomo-conv-48-q-196` delta `-999996` mode `temporal_evolution`: When did Deborah's parents give her first console?
- `locomo-conv-48-q-140` delta `-999995` mode `lexical_grounding`: Why did Jolene decide to get a snake as a pet?
- `locomo-conv-42-q-114` delta `-999994` mode `entity_exactness`: What is Nate's favorite genre of movies?
- `locomo-conv-48-q-051` delta `-999994` mode `entity_exactness`: Which year did Jolene start practicing yoga?
- `locomo-conv-44-q-048` delta `-999992` mode `gold_ambiguity`: How many months passed between Andrew adopting Buddy and Scout
- `locomo-conv-50-q-094` delta `-999992` mode `entity_exactness`: What is Dave's advice to Calvin regarding his dreams?
- `locomo-conv-44-q-035` delta `-999991` mode `gold_ambiguity`: How many months passed between Andrew adopting Toby and Buddy?
- `locomo-conv-48-q-050` delta `-999991` mode `temporal_evolution`: How long has Jolene been doing yoga and meditation?
- `locomo-conv-48-q-212` delta `-999991` mode `temporal_evolution`: For how long has Jolene had Lucifer as a pet?
- `locomo-conv-42-q-031` delta `-999990` mode `gold_ambiguity`: What kind of writings does Joanna do?

## Interpretation

BM25's lead is primarily a lexical/candidate-ranking signal. The comparison separates missing candidates from reranking regressions: `missing_candidate` means BM25 retrieved a gold memory in top 10 while Goncho did not; `rerank_regression` means both found gold but Goncho ranked it lower. Optimize only after reviewing these buckets.

Recommended next slice: inspect BM25-win `missing_candidate` and `rerank_regression` rows side by side with memory content, then decide whether candidate generation, metadata/noise, or conservative reranking needs work.

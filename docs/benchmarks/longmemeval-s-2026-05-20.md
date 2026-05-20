# LongMemEval-S Retrieval Run — 2026-05-20

This is a retrieval-only Goncho benchmark run against LongMemEval-S.

## Dataset

- Source: `xiaowu0162/longmemeval-cleaned` on Hugging Face.
- File: `longmemeval_s_cleaned.json`.
- Questions: 500.
- Converted memories: 23,867 haystack sessions.
- Conversion: one isolated Goncho peer per question; one memory per haystack session; gold IDs from `answer_session_ids`.
- Raw dataset and converted JSONL are not committed because they are large benchmark artifacts.

## Command

```sh
go run ./cmd/goncho-bench \
  --dataset ./artifacts/longmemeval/longmemeval-s-goncho.jsonl \
  --out ./artifacts/longmemeval/run20/report.json \
  --db ./artifacts/longmemeval/run20/bench.db \
  --limit 10 \
  --runs 20
```

Runtime evidence:

```text
elapsed=7:06.80 maxrss=614892KB
```

## Results

| System | Runs | R@5 strict | R@10 strict | recall_any@5 | recall_any@10 | MRR |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| Goncho BM25-style conclusion ranking | 20 | 88.90% | 93.86% | 96.40% | 98.00% | 81.12% |

`recall_any@K` is the metric used by the local research comparison table for LongMemEval retrieval. Strict `R@K` counts the fraction of all gold session IDs found, which is lower when a question has multiple gold sessions.

## Comparison to local research references

| System | recall_any@5 | recall_any@10 | MRR |
| --- | ---: | ---: | ---: |
| agentmemory BM25+Vector reference | 95.20% | 98.60% | 88.20% |
| agentmemory BM25-only reference | 86.20% | 94.60% | 71.50% |
| Goncho 2026-05-20 run | 96.40% | 98.00% | 81.12% |

## Interpretation

- Goncho beats the cited BM25-only reference on recall_any@5 and MRR.
- Goncho is slightly below the cited BM25-only reference on recall_any@10.
- Goncho beats the cited BM25+Vector reference on recall_any@5, but trails it on recall_any@10 and MRR.
- This is retrieval-only, not end-to-end QA with an LLM reader or judge.

## Validation after code changes

```sh
go test ./cmd/goncho-bench
go test ./...
go vet ./...
cd docs-site && npm run build
```

All commands passed after the run.

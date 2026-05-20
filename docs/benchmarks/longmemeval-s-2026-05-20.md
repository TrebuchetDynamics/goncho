# LongMemEval-S Retrieval Run — 2026-05-20

This report is generated from `docs/benchmarks/results/longmemeval-s-2026-05-20-goncho.json`. Treat the JSON file as the canonical evidence.

## Dataset

- Source: `xiaowu0162/longmemeval-cleaned` on Hugging Face.
- Revision: `98d7416c24c778c2fee6e6f3006e7a073259d48f`.
- SHA256: `d6f21ea9d60a0d56f34a05b609c79c88a451d2ae03597821ea3d5a9678c3a442`.
- Questions: 500.
- Converted memories: 23867 haystack sessions.
- Conversion: one isolated Goncho peer per question; one memory per haystack session; gold IDs from `answer_session_ids`.
- Raw dataset and converted JSONL are not committed because they are large benchmark artifacts.

## Environment

- Go: `go1.26.1`.
- OS/arch: `linux/amd64`.
- CPU count: 22.
- Runtime evidence: `elapsed=not recorded maxrss=not recorded`.

## Command

```sh
go run ./cmd/goncho-bench \
  --dataset ./artifacts/longmemeval/longmemeval-s-goncho.jsonl \
  --out docs/benchmarks/results/longmemeval-s-2026-05-20-goncho.json \
  --failures ./docs/benchmarks/failures/longmemeval-s-2026-05-20-goncho.jsonl \
  --db ./artifacts/longmemeval/goncho-science.db \
  --system goncho \
  --dataset-revision 98d7416c24c778c2fee6e6f3006e7a073259d48f \
  --dataset-sha256 d6f21ea9d60a0d56f34a05b609c79c88a451d2ae03597821ea3d5a9678c3a442 \
  --limit 10 \
  --runs 20
```

## Results

| System | Runs | R@5 strict | R@10 strict | recall_any@5 | recall_any@10 | MRR |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| goncho | 20 | 91.25% | 94.66% | 96.80% | 98.00% | 91.35% |

`recall_any@K` is the metric used by the local research comparison table for LongMemEval retrieval. Strict `R@K` counts the fraction of all gold session IDs found, which is lower when a question has multiple gold sessions.

## Leakage Checks

| Check | Count |
| --- | ---: |
| Exact query text present in indexed memory | 1 |
| Gold evidence IDs present in indexed memory content | 0 |

Examples:

- `0f05491a:query_in_memory:answer_d6d2eba8_1`

The one query-text hit in this run is an official LongMemEval case where the prior user message in the gold session exactly asks the later benchmark question. It is reported as leakage evidence instead of hidden.

## Comparison to local research references

| System | recall_any@5 | recall_any@10 | MRR |
| --- | ---: | ---: | ---: |
| agentmemory BM25+Vector reference | 95.20% | 98.60% | 88.20% |
| agentmemory BM25-only reference | 86.20% | 94.60% | 71.50% |
| Goncho 2026-05-20 run | 96.80% | 98.00% | 91.35% |

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

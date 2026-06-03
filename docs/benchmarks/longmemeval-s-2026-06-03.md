# LongMemEval-S Retrieval Run — 2026-06-03

This report summarizes `docs/benchmarks/results/longmemeval-s-2026-06-03-goncho.json`. Treat the JSON file as the canonical evidence.

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
- Runtime evidence: `elapsed=25:16.05 maxrss_kb=1117316`.

## Reproduction

Use the pinned full-run target to redownload/verify the dataset, convert it, and regenerate date-stamped reports for every deterministic backend:

```sh
make bench-longmemeval-s
```

The Goncho-only evidence below was produced with:

```sh
go run ./cmd/goncho-bench \
  --dataset ./artifacts/longmemeval/longmemeval-s-goncho.jsonl \
  --out ./docs/benchmarks/results/longmemeval-s-2026-06-03-goncho.json \
  --failures ./docs/benchmarks/failures/longmemeval-s-2026-06-03-goncho.jsonl \
  --db ./artifacts/longmemeval/goncho-2026-06-03.db \
  --system goncho \
  --dataset-revision 98d7416c24c778c2fee6e6f3006e7a073259d48f \
  --dataset-sha256 d6f21ea9d60a0d56f34a05b609c79c88a451d2ae03597821ea3d5a9678c3a442 \
  --limit 10 \
  --runs 20
```

For `--runs > 1`, the harness evaluates isolated temporary SQLite databases per run; the JSON and failure JSONL are the committed evidence artifacts.

## Results

| System | Runs | R@5 strict | R@10 strict | recall_any@5 | recall_any@10 | MRR |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| goncho | 20 | 91.12% | 94.66% | 96.80% | 98.00% | 91.22% |

`recall_any@K` is the LongMemEval-style any-gold-session retrieval metric. Strict `R@K` counts the fraction of all gold session IDs found, which is lower when a question has multiple gold sessions. MRR rewards ranking quality and is the harder signal for whether the first useful memory appears near the top of the result set.

## Leakage Checks

| Check | Count |
| --- | ---: |
| Exact query text present in indexed memory | 1 |
| Gold evidence IDs present in indexed memory content | 0 |

Examples:

- `0f05491a:query_in_memory:answer_d6d2eba8_1`

The one query-text hit is an official LongMemEval case where the prior user message in the gold session exactly asks the later benchmark question. It is reported as leakage evidence instead of hidden.

## Interpretation

- Goncho maintained `98.00%` recall_any@10 over a 23,867-memory corpus.
- Goncho reached `91.22%` MRR, meaning correct evidence usually appears near the top rather than merely somewhere in top-K.
- This is retrieval-only evidence, not end-to-end QA with an LLM reader or judge.

## Validation after code changes

```sh
go test ./cmd/goncho-bench
go test ./...
go vet ./...
cd docs-site && npm run build
```

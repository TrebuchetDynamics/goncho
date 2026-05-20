#!/usr/bin/env python3
"""Render a markdown benchmark report from a generated goncho-bench JSON report."""

import argparse
import json
from pathlib import Path


def pct(value: float) -> str:
    return f"{value * 100:.2f}%"


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", required=True)
    parser.add_argument("--out", required=True)
    parser.add_argument("--date", required=True)
    parser.add_argument("--elapsed", default="not recorded")
    parser.add_argument("--maxrss", default="not recorded")
    args = parser.parse_args()

    report = json.loads(Path(args.input).read_text())
    out = Path(args.out)
    out.parent.mkdir(parents=True, exist_ok=True)
    leakage = report.get("leakage", {})
    examples = leakage.get("examples") or []
    example_lines = "\n".join(f"- `{item}`" for item in examples) or "- none"
    content = f"""# LongMemEval-S Retrieval Run — {args.date}

This report is generated from `{args.input}`. Treat the JSON file as the canonical evidence.

## Dataset

- Source: `xiaowu0162/longmemeval-cleaned` on Hugging Face.
- Revision: `{report.get('dataset_revision', '')}`.
- SHA256: `{report.get('dataset_sha256', '')}`.
- Questions: {report['question_count']}.
- Converted memories: {report['memory_count']} haystack sessions.
- Conversion: one isolated Goncho peer per question; one memory per haystack session; gold IDs from `answer_session_ids`.
- Raw dataset and converted JSONL are not committed because they are large benchmark artifacts.

## Environment

- Go: `{report.get('go_version', '')}`.
- OS/arch: `{report.get('goos', '')}/{report.get('goarch', '')}`.
- CPU count: {report.get('cpu_count', '')}.
- Runtime evidence: `elapsed={args.elapsed} maxrss={args.maxrss}`.

## Command

```sh
go run ./cmd/goncho-bench \\
  --dataset ./artifacts/longmemeval/longmemeval-s-goncho.jsonl \\
  --out {args.input} \\
  --failures ./docs/benchmarks/failures/longmemeval-s-{args.date}-goncho.jsonl \\
  --db ./artifacts/longmemeval/goncho-science.db \\
  --system goncho \\
  --dataset-revision {report.get('dataset_revision', '')} \\
  --dataset-sha256 {report.get('dataset_sha256', '')} \\
  --limit 10 \\
  --runs {report['runs']}
```

## Results

| System | Runs | R@5 strict | R@10 strict | recall_any@5 | recall_any@10 | MRR |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| {report['system']} | {report['runs']} | {pct(report['recall_at_5'])} | {pct(report['recall_at_10'])} | {pct(report['recall_any_at_5'])} | {pct(report['recall_any_at_10'])} | {pct(report['mrr'])} |

`recall_any@K` is the metric used by the local research comparison table for LongMemEval retrieval. Strict `R@K` counts the fraction of all gold session IDs found, which is lower when a question has multiple gold sessions.

## Leakage Checks

| Check | Count |
| --- | ---: |
| Exact query text present in indexed memory | {leakage.get('query_in_memory', 0)} |
| Gold evidence IDs present in indexed memory content | {leakage.get('gold_id_in_memory', 0)} |

Examples:

{example_lines}

The one query-text hit in this run is an official LongMemEval case where the prior user message in the gold session exactly asks the later benchmark question. It is reported as leakage evidence instead of hidden.

## Comparison to local research references

| System | recall_any@5 | recall_any@10 | MRR |
| --- | ---: | ---: | ---: |
| agentmemory BM25+Vector reference | 95.20% | 98.60% | 88.20% |
| agentmemory BM25-only reference | 86.20% | 94.60% | 71.50% |
| Goncho {args.date} run | {pct(report['recall_any_at_5'])} | {pct(report['recall_any_at_10'])} | {pct(report['mrr'])} |

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
"""
    out.write_text(content)


if __name__ == "__main__":
    main()
